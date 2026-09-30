package doris

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/google/uuid"
)

const (
	defaultTableBaseName = "rethra_embeddings"
	envDorisTablePrefix  = "DORIS_TABLE_PREFIX"
)

// `NewDorisRetrieveEngineRepository`, Doris arama motoru deposunu oluşturur.
//
// Parametreler:
//   - `db`: MySQL protokolü kullanan `*sql.DB` örneği. `SetMaxOpenConns` gibi parametrelerden çağıran sorumludur.
//   - `feHTTPBase`: Stream Load için FE HTTP temel adresi (scheme dahil); örneğin `"http://doris-fe:8030"`.
//   - username/password: MySQL ve Stream Load tarafından ortak kullanılan kimlik bilgileri.
//   - database: Hedef veritabanı adı (hem MySQL DSN hem de Stream Load URL yolu için kullanılır).
//   - indexCfg: Boş olabilir. nil olduğunda ortam değişkenleri + varsayılan değerlere (env yolu) geri döner.
func NewDorisRetrieveEngineRepository(
	db *sql.DB,
	feHTTPBase, username, password, database string,
	indexCfg *types.IndexConfig,
) interfaces.RetrieveEngineRepository {
	log := logger.GetLogger(context.Background())
	log.Info("[Doris] Initializing Doris retriever engine repository")

	tableBaseName := types.ResolveCollectionName(indexCfg, envDorisTablePrefix, defaultTableBaseName)
	compatMode, invalidCompatMode := resolveConfiguredDorisCompatMode()
	if invalidCompatMode != "" {
		log.Warnf("[Doris] Invalid %s=%q, defaulting to %s", envDorisCompatMode, invalidCompatMode, dorisCompatModeAuto)
	}

	repo := &dorisRepository{
		db:                  db,
		httpClient:          newDorisStreamLoadHTTPClient(),
		feHTTPBase:          strings.TrimRight(feHTTPBase, "/"),
		username:            username,
		password:            password,
		database:            database,
		tableBaseName:       tableBaseName,
		bucketsNum:          indexCfg.GetBucketsNum(0),
		replicationNum:      indexCfg.GetReplicationNum(0),
		compatModeRequested: compatMode,
	}
	log.Infof("[Doris] Repository initialized: db=%s, base=%s, fe_http=%s, compat_mode=%s",
		database, tableBaseName, repo.feHTTPBase, repo.compatModeRequested)
	if os.Getenv(envDorisCompatMode) == "" {
		log.Infof("[Doris] %s not set, defaulting to %s and probing on first use", envDorisCompatMode, dorisCompatModeAuto)
	}
	return repo
}

func (r *dorisRepository) EngineType() types.RetrieverEngineType {
	return types.DorisRetrieverEngineType
}

func (r *dorisRepository) Support() []types.RetrieverType {
	return []types.RetrieverType{types.KeywordsRetrieverType, types.VectorRetrieverType}
}

// EstimateStorageSize, verilen IndexInfo listesinin depolama bayt sayısını tahmin eder.
//
// Qdrant algoritmasına bakın: payload alan uzunluğu + vektör baytları + HNSW komşuları + meta veriler.
func (r *dorisRepository) EstimateStorageSize(_ context.Context,
	indexInfoList []*types.IndexInfo, params map[string]any,
) int64 {
	var total int64
	for _, info := range indexInfoList {
		emb := toDorisVectorEmbedding(info, params, dorisCompatModeInnerProductDuplicate)
		total += calculateStorageSize(emb)
	}
	return total
}

// Save, tek bir kaydı karşılık gelen boyutun tablosuna yazar. Boş vektörler BatchSave içinde topluca reddedilir.
func (r *dorisRepository) Save(ctx context.Context,
	info *types.IndexInfo, additionalParams map[string]any,
) error {
	return r.BatchSave(ctx, []*types.IndexInfo{info}, additionalParams)
}

// BatchSave, aynı IndexInfo grubunu boyuta göre gruplar; Doris ANN uyumlu tablolar
// DUPLICATE KEY kullandığından, burada önceki değiştirme semantiğini korumak için id bazında açıkça delete + insert yapılır.
func (r *dorisRepository) BatchSave(ctx context.Context,
	indexInfoList []*types.IndexInfo, additionalParams map[string]any,
) error {
	log := logger.GetLogger(ctx)
	if len(indexInfoList) == 0 {
		return nil
	}
	compatMode, err := r.resolveCompatMode(ctx)
	if err != nil {
		return err
	}

	groups := make(map[int][]*DorisVectorEmbedding)
	for _, info := range indexInfoList {
		emb := toDorisVectorEmbedding(info, additionalParams, compatMode)
		if len(emb.Embedding) == 0 {
			log.Warnf("[Doris] Skipping empty embedding for chunk %s", info.ChunkID)
			continue
		}
		if err := validateEmbedding(emb.Embedding); err != nil {
			return fmt.Errorf("invalid embedding for chunk %s: %w", info.ChunkID, err)
		}
		// Kararlı bir birincil anahtar verin. SourceID, üst katmandaki en anlamlı "satır kimliği"dir,
		// ancak aynı chunk içinde birden fazla question olduğunda SourceID zaten benzersizdir, bu nedenle doğrudan kullanılır.
		if emb.ID == "" {
			emb.ID = emb.SourceID
		}
		if emb.ID == "" {
			emb.ID = uuid.New().String()
		}
		dim := len(emb.Embedding)
		groups[dim] = append(groups[dim], emb)
	}

	for dim, rows := range groups {
		if err := r.ensureTable(ctx, dim); err != nil {
			return err
		}
		if compatMode.usesReplaceWrite() {
			err = r.replaceRows(ctx, r.getTableName(dim), rows)
		} else {
			err = r.insertRows(ctx, r.getTableName(dim), rows)
		}
		if err != nil {
			return fmt.Errorf("batch save dim=%d: %w", dim, err)
		}
		log.Infof("[Doris] Saved %d rows to %s", len(rows), r.getTableName(dim))
	}
	return nil
}

// insertRows, sütun sırasına göre çoklu VALUES içeren bir INSERT oluşturur. embedding sütunu,
// go-sql-driver/mysql ARRAY yer tutucularını desteklemediğinden, SQL metnine değişmez değer olarak eklenmelidir.
func (r *dorisRepository) insertRows(ctx context.Context,
	table string, rows []*DorisVectorEmbedding,
) error {
	if len(rows) == 0 {
		return nil
	}

	// 9 normal yer tutucu + 1 embedding değişmez değeri.
	const perRowPlaceholders = "(?, ?, ?, ?, ?, ?, ?, ?, ?, %s)"

	parts := make([]string, len(rows))
	args := make([]any, 0, len(rows)*9)
	for i, e := range rows {
		parts[i] = fmt.Sprintf(perRowPlaceholders, embeddingLiteral(e.Embedding))
		args = append(args,
			e.ID, e.Content, e.SourceID, e.SourceType,
			e.ChunkID, e.KnowledgeID, e.KnowledgeBaseID, e.TagID,
			e.IsEnabled,
		)
	}

	stmt := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s",
		table,
		strings.Join(columns, ", "),
		strings.Join(parts, ", "),
	)
	_, err := r.db.ExecContext(ctx, stmt, args...)
	return err
}

// replaceRows, DUPLICATE KEY tablosunda "id'ye göre üzerine yazma" semantiğini açıkça taklit eder.
func (r *dorisRepository) replaceRows(ctx context.Context,
	table string, rows []*DorisVectorEmbedding,
) error {
	rows = dedupeRowsByID(rows)
	if len(rows) == 0 {
		return nil
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	if err := r.deleteRowsByID(ctx, table, ids); err != nil {
		return err
	}
	return r.insertRows(ctx, table, rows)
}

func (r *dorisRepository) deleteRowsByID(ctx context.Context,
	table string, ids []string,
) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE %s IN (%s)",
		table, fieldID, strings.Join(placeholders, ", "))
	_, err := r.db.ExecContext(ctx, stmt, args...)
	return err
}

func dedupeRowsByID(rows []*DorisVectorEmbedding) []*DorisVectorEmbedding {
	if len(rows) < 2 {
		return rows
	}

	out := make([]*DorisVectorEmbedding, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		if idx, ok := positions[row.ID]; ok {
			out[idx] = row
			continue
		}
		positions[row.ID] = len(out)
		out = append(out, row)
	}
	return out
}

// DeleteByChunkIDList, chunk_id sütununu kullanarak siler. dimension, belirli tabloyu bulmak için kullanılır.
func (r *dorisRepository) DeleteByChunkIDList(ctx context.Context,
	chunkIDList []string, dimension int, _ string,
) error {
	return r.deleteByField(ctx, fieldChunkID, chunkIDList, dimension)
}

// DeleteByKnowledgeIDList, knowledge_id sütununu kullanarak siler.
func (r *dorisRepository) DeleteByKnowledgeIDList(ctx context.Context,
	knowledgeIDList []string, dimension int, _ string,
) error {
	return r.deleteByField(ctx, fieldKnowledgeID, knowledgeIDList, dimension)
}

// DeleteBySourceIDList, source_id sütununu kullanarak siler.
func (r *dorisRepository) DeleteBySourceIDList(ctx context.Context,
	sourceIDList []string, dimension int, _ string,
) error {
	return r.deleteByField(ctx, fieldSourceID, sourceIDList, dimension)
}

// deleteByField, üç Delete* metodunun ortak uygulamasıdır:
// DELETE FROM <table> WHERE <field> IN (?, ?, ...)。
func (r *dorisRepository) deleteByField(ctx context.Context,
	field string, ids []string, dimension int,
) error {
	log := logger.GetLogger(ctx)
	if len(ids) == 0 {
		return nil
	}

	table := r.getTableName(dimension)
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, v := range ids {
		placeholders[i] = "?"
		args[i] = v
	}
	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE %s IN (%s)",
		table, field, strings.Join(placeholders, ", "))

	if _, err := r.db.ExecContext(ctx, stmt, args...); err != nil {
		log.Errorf("[Doris] Delete by %s failed: %v", field, err)
		return fmt.Errorf("delete by %s: %w", field, err)
	}
	log.Infof("[Doris] Deleted %d rows from %s by %s", len(ids), table, field)
	return nil
}

// Retrieve, RetrieverType değerine göre vektör aramasına veya anahtar kelime aramasına yönlendirir.
func (r *dorisRepository) Retrieve(ctx context.Context,
	params types.RetrieveParams,
) ([]*types.RetrieveResult, error) {
	switch params.RetrieverType {
	case types.VectorRetrieverType:
		return r.VectorRetrieve(ctx, params)
	case types.KeywordsRetrieverType:
		return r.KeywordsRetrieve(ctx, params)
	}
	return nil, fmt.Errorf("invalid retriever type: %v", params.RetrieverType)
}

// VectorRetrieve, sorgu vektörünü önce birimleştirir, ardından inner_product_approximate çağrısıyla
// ANN araması yapar; birim vektörler için inner product ile cosine similarity eşdeğerdir,
// Bu yüzden score hâlâ "büyük olan daha benzer" semantiğini korur.
func (r *dorisRepository) VectorRetrieve(ctx context.Context,
	params types.RetrieveParams,
) ([]*types.RetrieveResult, error) {
	log := logger.GetLogger(ctx)
	if err := validateEmbedding(params.Embedding); err != nil {
		return nil, fmt.Errorf("invalid query embedding: %w", err)
	}
	compatMode, err := r.resolveCompatMode(ctx)
	if err != nil {
		return nil, err
	}
	queryEmbedding := append([]float32(nil), params.Embedding...)
	if compatMode.normalizeEmbeddings() {
		queryEmbedding = normalizeEmbedding(queryEmbedding)
	}
	dim := len(params.Embedding)
	table := r.getTableName(dim)

	exists, err := r.tableExists(ctx, table)
	if err != nil {
		return nil, fmt.Errorf("check table %s: %w", table, err)
	}
	if !exists {
		log.Warnf("[Doris] Table %s does not exist, returning empty results", table)
		return buildRetrieveResult(nil, types.VectorRetrieverType), nil
	}

	wb := buildBaseFilter(params)
	whereClause, whereArgs := wb.build()

	// embedding değişmez değer olarak kullanılmalıdır; Doris, LIMIT/OFFSET için yer tutucuları desteklemez, değişmez değer olarak satır içine yazılmalıdır.
	// HAVING kullanılmasının nedeni score'un SELECT sütun takma adı olması ve WHERE aşamasında henüz görünmemesidir.
	scoreExpr := fmt.Sprintf("inner_product_approximate(`%s`, %s)", fieldEmbedding, embeddingLiteral(queryEmbedding))
	if compatMode == dorisCompatModeLegacy {
		scoreExpr = fmt.Sprintf("(1 - cosine_distance_approximate(`%s`, %s))", fieldEmbedding, embeddingLiteral(queryEmbedding))
	}
	stmt := fmt.Sprintf(
		"SELECT %s, %s AS score "+
			"FROM `%s` WHERE %s "+
			"HAVING score >= ? "+
			"ORDER BY score DESC LIMIT %d",
		strings.Join(columnsForRetrieve, ", "),
		scoreExpr,
		table,
		whereClause,
		params.TopK,
	)
	args := append(whereArgs, params.Threshold)

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, r.wrapVectorRetrieveError(table, compatMode, err)
	}
	defer rows.Close()

	results, err := scanRetrieveRows(rows, types.MatchTypeEmbedding)
	if err != nil {
		return nil, err
	}
	log.Infof("[Doris] Vector retrieval found %d results in %s", len(results), table)
	return buildRetrieveResult(results, types.VectorRetrieverType), nil
}

// KeywordsRetrieve, anahtar kelime eşleştirmesi için Doris ters dizini + MATCH_ANY kullanır.
//
// jieba istemci tarafı sözcük ayrıştırmasına gerek yoktur: CREATE TABLE sırasında idx_content zaten chinese parser olarak tanımlanmıştır.
// Farklı boyutlardaki tablolardan tablolar arası birleştirmeyle topK alınması, Milvus/Weaviate'ın mevcut durumuyla tutarlıdır.
func (r *dorisRepository) KeywordsRetrieve(ctx context.Context,
	params types.RetrieveParams,
) ([]*types.RetrieveResult, error) {
	log := logger.GetLogger(ctx)
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return buildRetrieveResult(nil, types.KeywordsRetrieverType), nil
	}

	tables, err := r.listEmbeddingTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	if len(tables) == 0 {
		return buildRetrieveResult(nil, types.KeywordsRetrieverType), nil
	}

	wb := buildBaseFilter(params)
	whereClause, whereArgs := wb.build()

	var all []*types.IndexWithScore
	for _, table := range tables {
		stmt := fmt.Sprintf(
			"SELECT %s FROM `%s` WHERE %s AND %s MATCH_ANY ? LIMIT %d",
			strings.Join(columnsForRetrieve, ", "),
			table, whereClause, fieldContent,
			params.TopK,
		)
		args := append(append([]any{}, whereArgs...), query)

		rows, err := r.db.QueryContext(ctx, stmt, args...)
		if err != nil {
			log.Warnf("[Doris] Keyword retrieve in %s failed: %v", table, err)
			continue
		}
		// KeywordsRetrieve içinde score sabit olarak 1.0'dır; bu, Qdrant davranışıyla tutarlıdır.
		batch, scanErr := scanRetrieveRows(rows, types.MatchTypeKeywords)
		_ = rows.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		all = append(all, batch...)
	}
	if len(all) > params.TopK {
		all = all[:params.TopK]
	}
	log.Infof("[Doris] Keywords retrieval found %d results across %d tables", len(all), len(tables))
	return buildRetrieveResult(all, types.KeywordsRetrieverType), nil
}

// CopyIndices, embedding'i yeniden oluşturmaktan kaçınmak için kaynak bilgi tabanındaki chunk'ları hedef bilgi tabanına kopyalar.
//
// Qdrant uygulamasıyla tamamen aynıdır:
//   - Kaynak tabloyu sayfalama ile tara
//   - chunk_id değerlerini sourceToTargetChunkIDMap ile çevir
//   - source_id çeviri kurallarını işle (normal chunk / üretici soru / diğerleri)
//   - Hedef satırları aynı tabloya geri yaz
func (r *dorisRepository) CopyIndices(ctx context.Context,
	sourceKnowledgeBaseID string,
	sourceToTargetKBIDMap map[string]string,
	sourceToTargetChunkIDMap map[string]string,
	targetKnowledgeBaseID string,
	dimension int,
	_ string,
) error {
	log := logger.GetLogger(ctx)
	if len(sourceToTargetChunkIDMap) == 0 {
		return nil
	}
	if err := r.ensureTable(ctx, dimension); err != nil {
		return err
	}

	table := r.getTableName(dimension)
	const pageSize = 64
	offset := 0
	totalCopied := 0

	for {
		stmt := fmt.Sprintf(
			"SELECT %s FROM `%s` WHERE %s = ? ORDER BY %s LIMIT %d OFFSET %d",
			strings.Join(columnsForCopy, ", "),
			table, fieldKnowledgeBaseID, fieldID,
			pageSize, offset,
		)
		rows, err := r.db.QueryContext(ctx, stmt, sourceKnowledgeBaseID)
		if err != nil {
			return fmt.Errorf("copy indices scan: %w", err)
		}
		batch, err := scanCopyRows(rows)
		_ = rows.Close()
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		var targets []*DorisVectorEmbedding
		for _, src := range batch {
			targetChunkID, ok := sourceToTargetChunkIDMap[src.ChunkID]
			if !ok {
				log.Warnf("[Doris] Source chunk %s not in target mapping", src.ChunkID)
				continue
			}
			targetKnowledgeID, ok := sourceToTargetKBIDMap[src.KnowledgeID]
			if !ok {
				log.Warnf("[Doris] Source knowledge %s not in target mapping", src.KnowledgeID)
				continue
			}

			targetSourceID := translateSourceID(src.SourceID, src.ChunkID, targetChunkID)
			targets = append(targets, &DorisVectorEmbedding{
				ID:              uuid.New().String(),
				Content:         src.Content,
				SourceID:        targetSourceID,
				SourceType:      src.SourceType,
				ChunkID:         targetChunkID,
				KnowledgeID:     targetKnowledgeID,
				KnowledgeBaseID: targetKnowledgeBaseID,
				TagID:           src.TagID,
				IsEnabled:       src.IsEnabled,
				Embedding:       src.Embedding,
			})
		}

		if len(targets) > 0 {
			if err := r.insertRows(ctx, table, targets); err != nil {
				return fmt.Errorf("copy indices insert: %w", err)
			}
			totalCopied += len(targets)
		}

		if len(batch) < pageSize {
			break
		}
		offset += pageSize
	}
	log.Infof("[Doris] CopyIndices done, dim=%d, copied=%d", dimension, totalCopied)
	return nil
}

// BatchUpdateChunkEnabledStatus / BatchUpdateChunkTagID'nin gerçek uygulaması streamload.go içinde bulunur,
// compat mode'a göre partial update veya rewrite rows seçilir.

// ---------------------------------------------------------------------------
// Özel yardımcı
// ---------------------------------------------------------------------------

// toDorisVectorEmbedding, IndexInfo + üst katmandan iletilen embedding eşlemesini şuna dönüştürür:
// Doris satır modeli. Embedding, additionalParams[fieldEmbedding] içindeki
// map[string][]float32 üzerinden SourceID ile alınır; Qdrant/Milvus ile tamamen aynıdır.
// inner_product_duplicate modu önce birimleştirir, legacy modu özgün vektörü korur.
func toDorisVectorEmbedding(
	info *types.IndexInfo,
	additionalParams map[string]any,
	compatMode dorisCompatMode,
) *DorisVectorEmbedding {
	emb := &DorisVectorEmbedding{
		ID:              info.ID,
		Content:         info.Content,
		SourceID:        info.SourceID,
		SourceType:      int(info.SourceType),
		ChunkID:         info.ChunkID,
		KnowledgeID:     info.KnowledgeID,
		KnowledgeBaseID: info.KnowledgeBaseID,
		TagID:           info.TagID,
		IsEnabled:       info.IsEnabled,
	}
	if additionalParams != nil {
		if v, ok := additionalParams[fieldEmbedding]; ok {
			if m, ok := v.(map[string][]float32); ok {
				emb.Embedding = append([]float32(nil), m[info.SourceID]...)
				if compatMode.normalizeEmbeddings() {
					emb.Embedding = normalizeEmbedding(emb.Embedding)
				}
			}
		}
	}
	return emb
}

func (r *dorisRepository) wrapVectorRetrieveError(table string, compatMode dorisCompatMode, err error) error {
	if compatMode == dorisCompatModeLegacy {
		return fmt.Errorf(
			"vector retrieve %s in Doris compat mode %s: %w. If your Doris build does not support cosine_distance_approximate or ANN on UNIQUE KEY tables, set %s=%s before creating embedding tables. %s is not interchangeable after %s_* tables are created",
			table,
			compatMode,
			err,
			envDorisCompatMode,
			dorisCompatModeInnerProductDuplicate,
			envDorisCompatMode,
			r.tableBaseName,
		)
	}
	return fmt.Errorf("vector retrieve %s in Doris compat mode %s: %w", table, compatMode, err)
}

// translateSourceID, kaynak SourceID'yi hedef SourceID'ye çevirir ve Qdrant uygulamasıyla tamamen aynıdır:
//   - Normal chunk: SourceID == ChunkID  -> targetChunkID kullanılır
//   - Üretici soru: SourceID == "<chunkID>-<questionID>" -> "<targetChunkID>-<questionID>"
//   - Diğer durumlar: yeni bir UUID oluşturulur (benzersizlik korunur)
func translateSourceID(originalSourceID, sourceChunkID, targetChunkID string) string {
	switch {
	case originalSourceID == sourceChunkID:
		return targetChunkID
	case strings.HasPrefix(originalSourceID, sourceChunkID+"-"):
		questionID := strings.TrimPrefix(originalSourceID, sourceChunkID+"-")
		return fmt.Sprintf("%s-%s", targetChunkID, questionID)
	default:
		return uuid.New().String()
	}
}

// scanRetrieveRows, Retrieve aşamasındaki rows'u IndexWithScore listesine seriden çıkarır.
//
// İki yola ayrılır:
//   - Sütun sayısı == columnsForRetrieve+1: N+1. sütun score'dur (vektör arama yolu)
//   - Sütun sayısı == columnsForRetrieve: score değerini tekdüze olarak 1.0 ata (anahtar kelime arama yolu)
func scanRetrieveRows(rows *sql.Rows, matchType types.MatchType) ([]*types.IndexWithScore, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	withScore := len(cols) == len(columnsForRetrieve)+1

	var out []*types.IndexWithScore
	for rows.Next() {
		var (
			id, content, sourceID, chunkID      string
			knowledgeID, knowledgeBaseID, tagID string
			sourceType                          int
			isEnabled                           bool
			score                               float64
			err                                 error
		)
		if withScore {
			err = rows.Scan(&id, &content, &sourceID, &sourceType,
				&chunkID, &knowledgeID, &knowledgeBaseID, &tagID, &isEnabled, &score)
		} else {
			err = rows.Scan(&id, &content, &sourceID, &sourceType,
				&chunkID, &knowledgeID, &knowledgeBaseID, &tagID, &isEnabled)
			score = 1.0
		}
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		out = append(out, &types.IndexWithScore{
			ID:              id,
			Content:         content,
			SourceID:        sourceID,
			SourceType:      types.SourceType(sourceType),
			ChunkID:         chunkID,
			KnowledgeID:     knowledgeID,
			KnowledgeBaseID: knowledgeBaseID,
			TagID:           tagID,
			Score:           score,
			MatchType:       matchType,
		})
	}
	return out, rows.Err()
}

// scanCopyRows, CopyIndices'in sayfalanmış sorgu sonucunu serileştirmeden çıkarır.
//
// scanRetrieveRows'tan farklı olarak burada embedding alanı gerekir (orijinal vektörü kopyalamak için).
// Doris'in ARRAY<FLOAT> değeri mysql protokolü üzerinden "[1,2,3]" dize değişmezi olarak döner.
func scanCopyRows(rows *sql.Rows) ([]*DorisVectorEmbedding, error) {
	var out []*DorisVectorEmbedding
	for rows.Next() {
		var (
			id, content, sourceID, chunkID      string
			knowledgeID, knowledgeBaseID, tagID string
			sourceType                          int
			isEnabled                           bool
			embeddingRaw                        sql.RawBytes
		)
		if err := rows.Scan(&id, &content, &sourceID, &sourceType,
			&chunkID, &knowledgeID, &knowledgeBaseID, &tagID, &isEnabled, &embeddingRaw); err != nil {
			return nil, fmt.Errorf("scan copy row: %w", err)
		}
		vec, err := parseEmbeddingLiteral(embeddingRaw)
		if err != nil {
			return nil, fmt.Errorf("parse embedding: %w", err)
		}
		out = append(out, &DorisVectorEmbedding{
			ID:              id,
			Content:         content,
			SourceID:        sourceID,
			SourceType:      sourceType,
			ChunkID:         chunkID,
			KnowledgeID:     knowledgeID,
			KnowledgeBaseID: knowledgeBaseID,
			TagID:           tagID,
			IsEnabled:       isEnabled,
			Embedding:       vec,
		})
	}
	return out, rows.Err()
}

// buildRetrieveResult, IndexWithScore listesini RetrieveResult içine sarar.
func buildRetrieveResult(results []*types.IndexWithScore, retrieverType types.RetrieverType) []*types.RetrieveResult {
	return []*types.RetrieveResult{{
		Results:             results,
		RetrieverEngineType: types.DorisRetrieverEngineType,
		RetrieverType:       retrieverType,
		Error:               nil,
	}}
}

// calculateStorageSize, tek bir satırın depolama maliyetini tahmin eder.
//
// Qdrant ile tutarlı olarak: payload dize baytları + vektör (dim*4) + HNSW M*2*8 + meta veri 24.
func calculateStorageSize(emb *DorisVectorEmbedding) int64 {
	var payload int64
	payload += int64(len(emb.Content))
	payload += int64(len(emb.SourceID))
	payload += int64(len(emb.ChunkID))
	payload += int64(len(emb.KnowledgeID))
	payload += int64(len(emb.KnowledgeBaseID))
	payload += int64(len(emb.TagID))
	payload += 8 // source_type int

	var vec int64
	var hnsw int64
	if len(emb.Embedding) > 0 {
		vec = int64(len(emb.Embedding)) * 4
		const hnswM = 32
		hnsw = hnswM * 2 * 8
	}
	const metaBytes int64 = 24
	return payload + vec + hnsw + metaBytes
}
