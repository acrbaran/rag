package doris

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/logger"
)

// Varsayılan kova sayısı / replikasyon sayısı. Doris, PROPERTIES içinde belirtilmediğinde küme varsayılan değerlerini kullanır,
// burada tek makine/küçük kümeler için daha uygun, temkinli bir değer verilir.
const (
	defaultBucketsNum     = 10
	defaultReplicationNum = 1

	// ANN dizini hazır olma yoklaması için azami bekleme süresi. Dizin hazır değilse yazma yolunu engellemez,
	// yalnızca ensureTable'ın kendisini engeller (ilk tablo oluşturma senaryosu), bu nedenle 30s kabul edilebilir.
	annReadyTimeout = 30 * time.Second
	annReadyPoll    = 1 * time.Second
)

// getTableName, belirli bir boyuta karşılık gelen fiziksel tablo adını döndürür: <base>_<dim>.
//
// Qdrant/Milvus/Weaviate'in collection adlandırma kuralıyla tutarlıdır,
// böylece farklı embedding modellerinin (farklı boyutların) verileri birbiriyle çakışmaz.
func (r *dorisRepository) getTableName(dimension int) string {
	return fmt.Sprintf("%s_%d", r.tableBaseName, dimension)
}

// ensureTable, hedef boyuta karşılık gelen tablonun zaten var olmasını garanti eder;
// yoksa CREATE TABLE IF NOT EXISTS ile oluşturur ve oluşturulduktan sonra ANN dizininin hazır olmasını yoklar.
//
// Bu yöntem her Save / BatchSave işleminden önce çağrılır; sonuç initializedTables içinde önbelleğe alınır,
// aynı süreçte aynı dimension için gerçekten yalnızca bir kez SHOW TABLES + DDL çalıştırılır.
func (r *dorisRepository) ensureTable(ctx context.Context, dimension int) error {
	if _, ok := r.initializedTables.Load(dimension); ok {
		return nil
	}
	compatMode, err := r.resolveCompatMode(ctx)
	if err != nil {
		return err
	}

	log := logger.GetLogger(ctx)
	tableName := r.getTableName(dimension)

	exists, err := r.tableExists(ctx, tableName)
	if err != nil {
		log.Errorf("[Doris] Failed to check table existence: %v", err)
		return fmt.Errorf("check table existence: %w", err)
	}

	if !exists {
		log.Infof("[Doris] Creating table %s with dimension %d in compat mode %s", tableName, dimension, compatMode)
		if err := r.createTable(ctx, tableName, dimension, compatMode); err != nil {
			log.Errorf("[Doris] Failed to create table: %v", err)
			return fmt.Errorf("create table: %w", err)
		}

		// ANN dizini Doris tarafında eşzamansız olarak oluşturulur. Burada arka plan goroutine içinde hazır olma durumu yoklanır,
		// yazma yolu engellenmez—dizin hazır olana kadar arama brute-force'a geriler (sonuçlar doğru, hız yavaş),
		// ilk yazma grubunun 30s takılmasına izin vermekten daha kabul edilebilirdir.
		go func(tn string) {
			// İstek düzeyindeki ctx iptalinin arka plan yoklamasını da durdurmaması için bağımsız bir context (timeout ile) kullanılır.
			bgCtx, cancel := context.WithTimeout(context.Background(), annReadyTimeout)
			defer cancel()
			if err := r.waitANNReady(bgCtx, tn); err != nil {
				logger.GetLogger(bgCtx).Warnf(
					"[Doris] ANN index for %s not ready within %s: %v "+
						"(queries may fall back to brute force temporarily)",
					tn, annReadyTimeout, err)
				return
			}
			logger.GetLogger(bgCtx).Infof("[Doris] ANN index for %s ready", tn)
		}(tableName)
	}

	r.initializedTables.Store(dimension, true)
	return nil
}

// tableExists, tablonun var olup olmadığını information_schema üzerinden belirler.
//
// SHOW TABLES doğrudan kullanılmaz; çünkü Doris 4.1'de SHOW TABLES LIKE büyük/küçük harfe duyarlıdır,
// information_schema ise MySQL ile daha iyi uyumludur.
func (r *dorisRepository) tableExists(ctx context.Context, tableName string) (bool, error) {
	const q = `SELECT COUNT(1) FROM information_schema.tables
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`
	var n int
	if err := r.db.QueryRowContext(ctx, q, r.database, tableName).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// createTable, CREATE TABLE DDL'sini gönderir. Doris DDL'si eşzamanlıdır (ANN indeks oluşturma hariç),
// başarı dönüşü tablonun yazılabilir olduğunu gösterir.
func (r *dorisRepository) createTable(ctx context.Context, tableName string, dimension int, compatMode dorisCompatMode) error {
	buckets := r.bucketsNum
	if buckets <= 0 {
		buckets = defaultBucketsNum
	}
	replication := r.replicationNum
	if replication <= 0 {
		replication = defaultReplicationNum
	}

	ddl := buildCreateTableDDL(tableName, dimension, buckets, replication, compatMode)
	_, err := r.db.ExecContext(ctx, ddl)
	if err != nil && compatMode == dorisCompatModeLegacy {
		return fmt.Errorf(
			"legacy Doris table creation failed: %w. If your Doris build rejects ANN indexes on UNIQUE KEY tables, set %s=%s before creating embedding tables. %s is not interchangeable after %s_* tables are created",
			err,
			envDorisCompatMode,
			dorisCompatModeInnerProductDuplicate,
			envDorisCompatMode,
			r.tableBaseName,
		)
	}
	return err
}

// buildCreateTableDDL, boyuta göre CREATE TABLE DDL'si oluşturur.
//
// Temel noktalar:
//   - DUPLICATE KEY(id): ANN indeksi için mevcut Doris/SelectDB tablo modeli gereksinimleriyle uyumludur.
//     Rethra, Go tarafında id'ye göre değiştirme yazma semantiğini korumak için delete + insert kullanır.
//   - INVERTED indeksi tüm filtre alanlarını ve Çince sözcük bölmeli content tam metin indeksini kapsar.
//   - ANN indeksi HNSW + inner_product kullanır; Doris, yazma/sorgulama öncesinde vektörleri birimleştirir,
//     bu nedenle genel olarak diğer vektör veritabanlarıyla tutarlı cosine benzerliği semantiğini korur.
//
// Not: DDL'deki dimension / buckets / replication adlı üç sayısal alan Go tarafında biçimlendirilerek birleştirilir,
// SQL enjeksiyonu riski yoktur (kaynakların tümü denetimli IndexConfig int değerleridir).
func buildCreateTableDDL(tableName string, dimension, buckets, replication int, compatMode dorisCompatMode) string {
	metricType := "inner_product"
	keyMode := "DUPLICATE KEY(id)"
	properties := fmt.Sprintf("\t\"replication_num\"=\"%d\"", replication)
	if compatMode == dorisCompatModeLegacy {
		metricType = "cosine_distance"
		keyMode = "UNIQUE KEY(id)"
		properties = fmt.Sprintf("\t\"replication_num\"=\"%d\",\n\t\"enable_unique_key_merge_on_write\"=\"true\"", replication)
	}

	const tpl = `CREATE TABLE IF NOT EXISTS ` + "`%s`" + ` (
    id                VARCHAR(64)  NOT NULL,
    chunk_id          VARCHAR(64),
    knowledge_id      VARCHAR(64),
    knowledge_base_id VARCHAR(64),
    source_id         VARCHAR(255),
    source_type       INT,
    tag_id            VARCHAR(64),
    is_enabled        BOOLEAN,
    content           TEXT,
    embedding         ARRAY<FLOAT> NOT NULL,
    INDEX idx_chunk    (chunk_id)          USING INVERTED,
    INDEX idx_kb       (knowledge_base_id) USING INVERTED,
    INDEX idx_kid      (knowledge_id)      USING INVERTED,
    INDEX idx_src      (source_id)         USING INVERTED,
    INDEX idx_tag      (tag_id)            USING INVERTED,
    INDEX idx_enabled  (is_enabled)        USING INVERTED,
    INDEX idx_content  (content)           USING INVERTED PROPERTIES("parser"="chinese","support_phrase"="true"),
    INDEX idx_emb      (embedding)         USING ANN PROPERTIES(
        "index_type"="hnsw",
		"metric_type"="%s",
        "dim"="%d",
        "max_degree"="32",
        "ef_construction"="200"
    )
) ENGINE=OLAP
%s
DISTRIBUTED BY HASH(id) BUCKETS %d
PROPERTIES(
	%s
);`
	return fmt.Sprintf(tpl, tableName, metricType, dimension, keyMode, buckets, properties)
}

// waitANNReady, ANN indeksinin FINISHED durumuna girmesini beklemek için SHOW INDEX'i sorgular.
//
// Doris'in ANN indeksi tablo oluşturulduktan sonra eşzamansız olarak kurulur; bu sırada sorgular brute-force'a düşer (sonuçlar doğru, hız düşüktür).
// Burada yalnızca "en iyi çaba" ile beklenir: süre dolduğunda hazır değilse yalnızca warning kaydedilir, yazma engellenmez.
func (r *dorisRepository) waitANNReady(ctx context.Context, tableName string) error {
	deadline := time.Now().Add(annReadyTimeout)
	for {
		ready, err := r.annIndexReady(ctx, tableName)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("ann index not ready within %s", annReadyTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(annReadyPoll):
		}
	}
}

// annIndexReady, ANN indeksinin State değerinin FINISHED olup olmadığını denetler.
//
// SHOW INDEX FROM <table>, Doris'te birden çok sütun döndürür; farklı küçük sürümlerde sütun sırası biraz değişir,
// burada sütun adı eşleştirmesi kullanılır (information_schema.statistics + özel view uygulanabilir değildir,
// doğrudan SHOW INDEX kullanıp taramak yeterlidir).
//
// Uyumluluk stratejisi: SHOW INDEX dönüşünde idx_emb satırı bulunamazsa (çok eski sürümler), hazır kabul edilir,
// farklı Doris sürümlerinin çıktı farklılıkları nedeniyle başlangıcın kilitlenmesi önlenir.
func (r *dorisRepository) annIndexReady(ctx context.Context, tableName string) (bool, error) {
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf("SHOW INDEX FROM `%s`", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return false, err
	}
	keyNameIdx, stateIdx := -1, -1
	for i, c := range cols {
		switch strings.ToLower(c) {
		case "key_name":
			keyNameIdx = i
		case "state", "index_state":
			stateIdx = i
		}
	}

	for rows.Next() {
		// Farklı sütun türleriyle uyumluluk için sql.RawBytes kullanılır.
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return false, err
		}

		var keyName, state string
		if keyNameIdx >= 0 {
			keyName = bytesToString(raw[keyNameIdx])
		}
		if stateIdx >= 0 {
			state = bytesToString(raw[stateIdx])
		}

		if keyName != "idx_emb" {
			continue
		}
		if stateIdx < 0 {
			// Eski sürümler state sütununu sunmaz; iyimser biçimde hazır kabul edilir.
			return true, nil
		}
		if !strings.EqualFold(state, "FINISHED") &&
			!strings.EqualFold(state, "NORMAL") {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	// Buraya gelinmesinin iki durumu vardır:
	//   1. idx_emb satırı bulunmuştur ve state zaten FINISHED/NORMAL'dir (veya stateIdx<0 olan eski sürüm);
	//   2. idx_emb satırı bulunamamıştır (çok eski Doris bu indeks adını sunmaz);
	// Hepsi hazır kabul edilir, engellenmez. Hazır olmayan dallar döngü içinde önceden return false yapmıştır.
	return true, nil
}

// listEmbeddingTables, mevcut database altındaki tüm <base>_% adlı tabloları döndürür,
// anahtar kelime araması / boyutlar arası BatchUpdate için kullanılır.
func (r *dorisRepository) listEmbeddingTables(ctx context.Context) ([]string, error) {
	const q = `SELECT TABLE_NAME FROM information_schema.tables
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME LIKE ?`
	rows, err := r.db.QueryContext(ctx, q, r.database, r.tableBaseName+"\\_%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// bytesToString, SHOW INDEX tarafından döndürülen raw any değerini (genellikle []byte veya string)
// güvenli biçimde dizeye dönüştürür.
func bytesToString(v any) string {
	switch s := v.(type) {
	case []byte:
		return string(s)
	case string:
		return s
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}
