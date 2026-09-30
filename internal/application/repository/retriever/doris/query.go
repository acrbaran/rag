package doris

import (
	"math"
	"strconv"
	"strings"

	"github.com/acrbaran/rag/internal/types"
)

// Alan adı sabitleri. Doris bir SQL kütüphanesidir; alan adları SELECT/WHERE/INSERT içinde birçok yerde yeniden kullanılır,
// Sabitleri kullanarak yazım hatalarını önle.
const (
	fieldID              = "id"
	fieldContent         = "content"
	fieldSourceID        = "source_id"
	fieldSourceType      = "source_type"
	fieldChunkID         = "chunk_id"
	fieldKnowledgeID     = "knowledge_id"
	fieldKnowledgeBaseID = "knowledge_base_id"
	fieldTagID           = "tag_id"
	fieldIsEnabled       = "is_enabled"
	fieldEmbedding       = "embedding"
)

// columns, INSERT / SELECT sırasında kullanılan standart sütun sırasıdır.
var columns = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled, fieldEmbedding,
}

// columnsForRetrieve, Retrieve sırasında SELECT için kullanılan sütun sırasıdır,
// embedding içermez (vektörün kendisinin sorgu sonucunda döndürülmesi gerekmez; bant genişliği tasarrufu sağlar).
var columnsForRetrieve = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled,
}

// columnsForCopy, CopyIndices içindeki sayfalı SELECT sırasında kullanılan sütun sırasıdır,
// columnsForRetrieve değerinden embedding kadar fazladır; çünkü kopyalamanın amacı vektörün kendisini taşımaktır.
var columnsForCopy = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled, fieldEmbedding,
}

// whereCond bir WHERE alt koşulunu temsil eder: clause parametreli SQL parçasıdır (? yer tutucusu içerir),
// args karşılık gelen sıradaki parametre değerleridir. Tüm kullanıcı girdisi alanları (IDs) args üzerinden aktarılmalıdır,
// Bunları clause dizgesine eklemek kesinlikle yasaktır.
type whereCond struct {
	clause string
	args   []any
}

// whereBuilder, RetrieveParams içindeki filtre koşullarını SQL WHERE alt cümlelerine dönüştürmek için kullanılır.
//
// Her add* yöntemi bir IN / NOT IN / = işleç türüne karşılık gelir; son build() işlemi bunları AND ile birleştirir.
type whereBuilder struct {
	conds []whereCond
}

// addEqual bir field = ? koşulu ekler.
func (w *whereBuilder) addEqual(field string, value any) {
	w.conds = append(w.conds, whereCond{
		clause: field + " = ?",
		args:   []any{value},
	})
}

// addIn bir field IN (?, ?, ...) koşulu ekler. values boşsa hiçbir şey eklenmez.
func (w *whereBuilder) addIn(field string, values []string) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	w.conds = append(w.conds, whereCond{
		clause: field + " IN (" + strings.Join(placeholders, ", ") + ")",
		args:   args,
	})
}

// addNotIn bir field NOT IN (?, ?, ...) koşulu ekler.
func (w *whereBuilder) addNotIn(field string, values []string) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	w.conds = append(w.conds, whereCond{
		clause: field + " NOT IN (" + strings.Join(placeholders, ", ") + ")",
		args:   args,
	})
}

// build, WHERE cümlesini ("WHERE " öneki olmadan) ve parametre dizisini döndürür.
// Hiç koşul olmadığında ("1 = 1", nil) döndürür; böylece çağıran taraf doğrudan birleştirebilir.
func (w *whereBuilder) build() (string, []any) {
	if len(w.conds) == 0 {
		return "1 = 1", nil
	}
	parts := make([]string, len(w.conds))
	var args []any
	for i, c := range w.conds {
		parts[i] = c.clause
		args = append(args, c.args...)
	}
	return strings.Join(parts, " AND "), args
}

// buildBaseFilter, RetrieveParams içindeki filtre koşullarını whereBuilder'a dönüştürür.
// Varsayılan olarak `is_enabled = TRUE` eklenir; Qdrant/Milvus/Weaviate ile tutarlıdır:
// Devre dışı bırakılan chunk'lar aramaya katılmaz.
func buildBaseFilter(params types.RetrieveParams) *whereBuilder {
	w := &whereBuilder{}
	w.addEqual(fieldIsEnabled, true)

	if len(params.KnowledgeBaseIDs) > 0 {
		w.addIn(fieldKnowledgeBaseID, params.KnowledgeBaseIDs)
	}
	if len(params.KnowledgeIDs) > 0 {
		w.addIn(fieldKnowledgeID, params.KnowledgeIDs)
	}
	if len(params.TagIDs) > 0 {
		w.addIn(fieldTagID, params.TagIDs)
	}
	if len(params.ExcludeKnowledgeIDs) > 0 {
		w.addNotIn(fieldKnowledgeID, params.ExcludeKnowledgeIDs)
	}
	if len(params.ExcludeChunkIDs) > 0 {
		w.addNotIn(fieldChunkID, params.ExcludeChunkIDs)
	}
	return w
}

// `parseEmbeddingLiteral`, MySQL protokolü üzerinden Doris `ARRAY<FLOAT>` olarak döndürülen değeri ayrıştırır
// `"[1,2,3]"` biçimindeki literal dizgeyi `[]float32` değerine dönüştürür.
//
// `CopyIndices` yolu, vektörün kendisini kaynak satırdan okuyup hedef satıra yeniden yazmalıdır; burada hata toleranslı ayrıştırma önceliklidir:
// `[]` olmadan da kabul edilir, boş dizi `nil` döndürür.
func parseEmbeddingLiteral(raw []byte) ([]float32, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return nil, nil
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]float32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := strconv.ParseFloat(p, 32)
		if err != nil {
			return nil, err
		}
		out = append(out, float32(f))
	}
	return out, nil
}

// `validateEmbedding`, vektör öğelerinin tümünün sonlu değerler olduğunu doğrular.
//
// `strconv.FormatFloat`, NaN/±Inf için `"NaN"`/`"+Inf"`/`"-Inf"` çıktısı üretir,
// bu literaller Doris tarafından oluşturulan SQL'de sözdizimi hatasına yol açar (veya bazı sürümlerde tanımsız sonuçlar üretir).
// Normal koşullarda üst katman (gömme modeli) sonlu olmayan değerler üretmez; ancak GPU OOM, üst katman hataları,
// ve test dublörleri bunu tetikleyebilir; burada hızlıca hata vermek, sessizce kirli veri yazmaktan daha güvenlidir.
func validateEmbedding(vec []float32) error {
	for i, v := range vec {
		if f := float64(v); math.IsNaN(f) || math.IsInf(f, 0) {
			return errInvalidEmbedding{index: i, value: v}
		}
	}
	return nil
}

// normalizeEmbedding returns a unit-length copy of vec so Doris inner-product
// ANN search can preserve cosine-style similarity semantics.
func normalizeEmbedding(vec []float32) []float32 {
	if len(vec) == 0 {
		return nil
	}
	var sumSquares float64
	for _, value := range vec {
		f := float64(value)
		sumSquares += f * f
	}
	if sumSquares == 0 {
		return append([]float32(nil), vec...)
	}
	norm := float32(math.Sqrt(sumSquares))
	normalized := make([]float32, len(vec))
	for i, value := range vec {
		normalized[i] = value / norm
	}
	return normalized
}

// `errInvalidEmbedding`, hangi indeksin sonlu olmayan bir değer içerdiğini belirtir; `fmt.Errorf` yerine yapı kullanılmasının nedeni
// üst katmanın sorun tespiti için günlüklerde indeksi alabilmesidir.
type errInvalidEmbedding struct {
	index int
	value float32
}

func (e errInvalidEmbedding) Error() string {
	return "doris: embedding[" + strconv.Itoa(e.index) +
		"] is not finite: " + strconv.FormatFloat(float64(e.value), 'g', -1, 32)
}

// `embeddingLiteral`, `[]float32` değerini Doris `ARRAY<FLOAT>` literal dizgesine dönüştürür:
// "[1.23,4.56,...]"。
//
// Neden yer tutucu kullanılmıyor: `go-sql-driver/mysql`, `ARRAY` türü için parametre bağlamayı desteklemez,
// Doris tarafı da yalnızca literal biçimini kabul eder. Burada `strconv.FormatFloat` (`'g'` + `bitSize=32`) kullanılır
// ve `fmt.Sprintf("%f", v)` kullanılmaz; bunun iki nedeni vardır:
//  1. `fmt`, bazı yerel ayarlarda binlik ayırıcı kullanarak SQL sözdizimini bozar;
//  2. `'g'`, `'f'` biçiminden daha kısadır ve hassasiyet kaybına yol açmaz.
//
// Enjeksiyon riski: `[]float32` öğeleri, gömme modeli tarafından üretilen sonlu bit sayılı kayan noktalı değerlerdir; serileştirildikten sonra yalnızca
// `[0-9eE+-.\s]` karakterlerini içerebilir ve literal bağlamından kaçamaz.
func embeddingLiteral(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.Grow(len(vec) * 12)
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}
