package doris

import (
	"database/sql"
	"net/http"
	"sync"
)

// dorisRepository, Apache Doris 4.1 için arama motoru depo uygulamasıdır.
//
// İletişim kanalları:
//   - Okuma-yazma ana hattı: MySQL protokolü (database/sql + go-sql-driver/mysql), FE varsayılan olarak 9030 portunu kullanır.
//   - Stream Load: HTTP (FE varsayılan olarak 8030 portu), legacy modu partial update için kullanılır.
//
// Tablo yapısı boyutlara göre ayrılır: <tableBaseName>_<dim>.
// Uyumluluk modu DORIS_COMPAT_MODE tarafından belirlenir:
//   - legacy：UNIQUE KEY(id) + cosine_distance ANN + Stream Load partial update
//   - inner_product_duplicate：DUPLICATE KEY(id) + normalized inner product + delete/insert rewrite
//
// Bu ayar, embedding tabloları oluşturulduktan sonra doğrudan değiştirilemez; mod değiştirilmeden önce bu tablolar yeniden oluşturulmalıdır.
//
// Qdrant/Milvus/Weaviate ile aynı şekilde, initializedTables "varlığı garanti edilmiş" boyutları önbelleğe alır,
// her yazmada SHOW TABLES çalıştırılmasını önler.
type dorisRepository struct {
	db *sql.DB

	httpClient *http.Client
	// fe HTTP tabanı, örneğin "http://doris-fe:8030". Stream Load yolu
	// streamLoadURL(table) ile birleştirilir: <feHTTPBase>/api/<database>/<table>/_stream_load.
	feHTTPBase string

	username string
	password string
	database string

	tableBaseName       string
	bucketsNum          int // 0 -> default 10
	replicationNum      int // 0 -> default 1
	compatModeRequested dorisCompatMode
	compatModeResolved  dorisCompatMode
	compatResolveOnce   sync.Once
	compatResolveErr    error

	// ensureTable işlemi zaten garanti edilmiş boyut kümesi: dim -> true.
	initializedTables sync.Map
}

// DorisVectorEmbedding, Doris tablosundaki bir satırın alan modeli olarak kullanılır.
//
// Alan sırası, schema.go içindeki INSERT sütun sırasıyla tutarlıdır,
// değiştirirken createInsert ve columns da birlikte güncellenmelidir.
type DorisVectorEmbedding struct {
	ID              string
	Content         string
	SourceID        string
	SourceType      int
	ChunkID         string
	KnowledgeID     string
	KnowledgeBaseID string
	TagID           string
	IsEnabled       bool
	Embedding       []float32
}

// DorisVectorEmbeddingWithScore, arama sonuçlarının alan modelidir,
// Score, vektör aramasında mevcut compat mode'a göre hesaplanır; anahtar kelime aramasında ise her zaman 1.0 atanır.
type DorisVectorEmbeddingWithScore struct {
	DorisVectorEmbedding
	Score float64
}
