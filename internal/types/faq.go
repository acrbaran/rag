package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/acrbaran/rag/internal/textconv"
)

// FAQChunkMetadata, FAQ öğelerinin Chunk.Metadata içindeki yapısını tanımlar
type FAQChunkMetadata struct {
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions,omitempty"`
	NegativeQuestions []string       `json:"negative_questions,omitempty"`
	Answers           []string       `json:"answers,omitempty"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy,omitempty"`
	Version           int            `json:"version,omitempty"`
	Source            string         `json:"source,omitempty"`
}

// GeneratedQuestion, AI tarafından oluşturulan tek bir soruyu temsil eder
type GeneratedQuestion struct {
	ID              string `json:"id"`                         // source_id oluşturmak için kullanılan benzersiz tanımlayıcı
	Question        string `json:"question"`                   // Soru içeriği
	ContentRevision *int   `json:"content_revision,omitempty"` // Bu soruya karşılık gelen Chunk içerik sürümü
}

const maxGeneratedQuestionSourceIDLength = 64

// GeneratedQuestionSourceID builds the retrieval source identifier for a
// generated question. PostgreSQL stores source_id as varchar(64), while a
// chunk UUID plus a question UUID would be 73 bytes. Preserve the historical
// representation for short IDs and hash only oversized question IDs so
// existing index rows remain addressable by delete/reindex operations.
func GeneratedQuestionSourceID(chunkID, questionID string) string {
	candidate := chunkID + "-" + questionID
	if len(candidate) <= maxGeneratedQuestionSourceIDLength {
		return candidate
	}
	digest := sha256.Sum256([]byte(questionID))
	// UUID chunk IDs use 36 bytes; "-q" plus 24 hex characters keeps the
	// complete identifier at 62 bytes while retaining ample collision space.
	return chunkID + "-q" + hex.EncodeToString(digest[:12])
}

// DocumentChunkMetadata, belge Chunk'ının meta veri yapısını tanımlar
// AI tarafından oluşturulan sorular gibi zenginleştirilmiş bilgileri depolamak için kullanılır
type DocumentChunkMetadata struct {
	// GeneratedQuestions, AI'ın bu Chunk için oluşturduğu ilgili soruları depolar
	// Bu sorular geri çağırma oranını artırmak için bağımsız olarak indekslenir
	GeneratedQuestions []GeneratedQuestion `json:"generated_questions,omitempty"`
	// GeneratedQuestionsRevision ties the questions to Chunk.ContentRevision.
	GeneratedQuestionsRevision int `json:"generated_questions_revision,omitempty"`
}

// IsQuestionCurrent reports whether a generated question was authored for the
// current chunk body. This is advisory metadata for the UI: questions remain
// valid retrieval aliases across chunk edits. Legacy rows fall back to the
// metadata-level revision.
func (m *DocumentChunkMetadata) IsQuestionCurrent(question GeneratedQuestion, chunkRevision int) bool {
	if question.ContentRevision != nil {
		return *question.ContentRevision == chunkRevision
	}
	return m != nil && m.GeneratedQuestionsRevision == chunkRevision
}

// GetQuestionStrings, soru içeriği dizeleri listesini döndürür (eski kodla uyumlu)
func (m *DocumentChunkMetadata) GetQuestionStrings() []string {
	if m == nil || len(m.GeneratedQuestions) == 0 {
		return nil
	}
	result := make([]string, len(m.GeneratedQuestions))
	for i, q := range m.GeneratedQuestions {
		result[i] = q.Question
	}
	return result
}

// DocumentMetadata, Chunk içindeki belge meta verilerini ayrıştırır
func (c *Chunk) DocumentMetadata() (*DocumentChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta DocumentChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SetDocumentMetadata, Chunk'ın belge meta verilerini ayarlar
func (c *Chunk) SetDocumentMetadata(meta *DocumentChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		return nil
	}
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	return nil
}

// Sanitize, meta veriler üzerinde temel temizlik yapar (baş ve son boşlukları kaldırma, yinelenenleri kaldırma) ve özgün içeriği korur
// DB depolaması için kullanılır, anlamsal normalleştirme yapılmaz
func (m *FAQChunkMetadata) Sanitize() {
	if m == nil {
		return
	}
	m.StandardQuestion = strings.TrimSpace(m.StandardQuestion)
	m.SimilarQuestions = SanitizeStrings(m.SimilarQuestions)
	m.NegativeQuestions = SanitizeStrings(m.NegativeQuestions)
	m.Answers = SanitizeStrings(m.Answers)
	if m.Version <= 0 {
		m.Version = 1
	}
}

// Normalize, Hash hesaplaması ve vektör indeksleme için normalleştirilmiş bir kopya döndürür
// Özgün veri değişmeden kalır, yeni bir normalleştirilmiş kopya döndürülür
func (m *FAQChunkMetadata) Normalize() *FAQChunkMetadata {
	if m == nil {
		return nil
	}
	return &FAQChunkMetadata{
		StandardQuestion:  NormalizeQuestion(m.StandardQuestion),
		SimilarQuestions:  normalizeQuestionStrings(m.SimilarQuestions),
		NegativeQuestions: normalizeQuestionStrings(m.NegativeQuestions),
		Answers:           SanitizeStrings(m.Answers), // Yanıt yalnızca temel temizliğe tabi tutulur
		AnswerStrategy:    m.AnswerStrategy,
		Version:           m.Version,
		Source:            m.Source,
	}
}

// SanitizeStrings, dize listesi üzerinde temel temizlik yapar (TrimSpace + yinelenenleri kaldırma)
func SanitizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	dedup := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		dedup = append(dedup, trimmed)
	}
	if len(dedup) == 0 {
		return nil
	}
	return dedup
}

// FAQMetadata, Chunk içindeki FAQ meta verilerini ayrıştırır
// Özgün veriyi döndürür (yalnızca temel temizlik yapılır)
func (c *Chunk) FAQMetadata() (*FAQChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta FAQChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	meta.Sanitize() // Yalnızca temel temizlik yapılır, özgün içerik korunur
	return &meta, nil
}

// SetFAQMetadata, Chunk'ın FAQ meta verilerini ayarlar
// DB ham verileri depolar, ContentHash normalize edilmiş verilere göre hesaplanır
func (c *Chunk) SetFAQMetadata(meta *FAQChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		c.ContentHash = ""
		return nil
	}
	// Temel temizlemeden sonra DB'ye depolanır (orijinal içerik korunur)
	meta.Sanitize()
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	// ContentHash, yinelenenleri kaldırma eşleştirmesi için normalize edilmiş verilere göre hesaplanır
	normalized := meta.Normalize()
	c.ContentHash = CalculateFAQContentHash(normalized)
	return nil
}

// CalculateFAQContentHash, FAQ içeriğinin hash değerini hesaplar
// hash şunlara dayanır: standart soru + benzer sorular (sıralandıktan sonra) + karşı örnekler (sıralandıktan sonra) + yanıtlar (sıralandıktan sonra)
// Hızlı eşleştirme ve yinelenenleri kaldırma için kullanılır
func CalculateFAQContentHash(meta *FAQChunkMetadata) string {
	if meta == nil {
		return ""
	}

	// Normalize() returns a new copy; the old code discarded the return value.
	normalized := meta.Normalize()
	if normalized == nil {
		return ""
	}

	// Dizileri sıralar (aynı içeriğin aynı hash'i üretmesini sağlar)
	similarQuestions := make([]string, len(normalized.SimilarQuestions))
	copy(similarQuestions, normalized.SimilarQuestions)
	sort.Strings(similarQuestions)

	negativeQuestions := make([]string, len(normalized.NegativeQuestions))
	copy(negativeQuestions, normalized.NegativeQuestions)
	sort.Strings(negativeQuestions)

	answers := make([]string, len(normalized.Answers))
	copy(answers, normalized.Answers)
	sort.Strings(answers)

	// hash için kullanılacak dizeyi oluşturur: standart soru + benzer sorular + karşı örnekler + yanıtlar
	var builder strings.Builder
	builder.WriteString(normalized.StandardQuestion)
	builder.WriteString("|")
	builder.WriteString(strings.Join(similarQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(negativeQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(answers, ","))

	// SHA256 hash'ini hesaplar
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// AnswerStrategy, yanıt döndürme stratejisini tanımlar
type AnswerStrategy string

const (
	// AnswerStrategyAll tüm yanıtları döndürür
	AnswerStrategyAll AnswerStrategy = "all"
	// AnswerStrategyRandom rastgele bir yanıt döndürür
	AnswerStrategyRandom AnswerStrategy = "random"
)

// FAQEntry, ön uca döndürülen FAQ öğesini temsil eder
type FAQEntry struct {
	ID                int64          `json:"id"`
	ChunkID           string         `json:"chunk_id"`
	KnowledgeID       string         `json:"knowledge_id"`
	KnowledgeBaseID   string         `json:"knowledge_base_id"`
	TagID             int64          `json:"tag_id"`
	TagName           string         `json:"tag_name"`
	IsEnabled         bool           `json:"is_enabled"`
	IsRecommended     bool           `json:"is_recommended"`
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions"`
	NegativeQuestions []string       `json:"negative_questions"`
	Answers           []string       `json:"answers"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy"`
	IndexMode         FAQIndexMode   `json:"index_mode"`
	UpdatedAt         time.Time      `json:"updated_at"`
	CreatedAt         time.Time      `json:"created_at"`
	Score             float64        `json:"score,omitempty"`
	MatchType         MatchType      `json:"match_type,omitempty"`
	ChunkType         ChunkType      `json:"chunk_type"`
	// MatchedQuestion is the actual question text that was matched in FAQ search
	// Could be the standard question or one of the similar questions
	MatchedQuestion string `json:"matched_question,omitempty"`
}

// FAQExportEntry, JSON olarak dışa aktarılan FAQ öğesini temsil eder ve FAQEntryPayload içe aktarma biçimiyle uyumludur,
// "dışa aktarma → düzenleme → yeniden içe aktarma" döngüsünü kolaylaştırır. Yeni alanlar eklerken mutlaka omitempty'yi koruyun, aksi halde
// geçmiş dışa aktarma dosyalarının uyumluluğu bozulur.
type FAQExportEntry struct {
	ID                int64          `json:"id"`
	TagName           string         `json:"tag_name,omitempty"`
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions,omitempty"`
	NegativeQuestions []string       `json:"negative_questions,omitempty"`
	Answers           []string       `json:"answers,omitempty"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy,omitempty"`
	IsEnabled         bool           `json:"is_enabled"`
	IsRecommended     bool           `json:"is_recommended"`
}

// FAQEntryPayload, FAQ öğeleri oluşturmak/güncellemek için kullanılan payload'dur
type FAQEntryPayload struct {
	// ID isteğe bağlıdır; veri aktarımı sırasında seq_id belirtmek için kullanılır (otomatik artış başlangıç değeri olan 100000000'den küçük olmalıdır)
	ID                *int64          `json:"id,omitempty"`
	StandardQuestion  string          `json:"standard_question"    binding:"required"`
	SimilarQuestions  []string        `json:"similar_questions"`
	NegativeQuestions []string        `json:"negative_questions"`
	Answers           []string        `json:"answers"`
	AnswerStrategy    *AnswerStrategy `json:"answer_strategy,omitempty"`
	TagID             int64           `json:"tag_id"`
	TagName           string          `json:"tag_name"`
	IsEnabled         *bool           `json:"is_enabled,omitempty"`
	IsRecommended     *bool           `json:"is_recommended,omitempty"`
}

const (
	FAQBatchModeAppend  = "append"
	FAQBatchModeReplace = "replace"
)

// FAQBatchUpsertPayload, FAQ öğelerini toplu olarak içe aktarır
type FAQBatchUpsertPayload struct {
	Entries     []FAQEntryPayload `json:"entries"      binding:"required"`
	Mode        string            `json:"mode"         binding:"oneof=append replace"`
	KnowledgeID string            `json:"knowledge_id"`
	TaskID      string            `json:"task_id"` // İsteğe bağlıdır, belirtilmezse UUID otomatik oluşturulur
	DryRun      bool              `json:"dry_run"` // Yalnızca doğrular, gerçekten içe aktarma yapmaz
}

// FAQFailedEntry, içe aktarma/doğrulama başarısız olan öğeyi temsil eder
type FAQFailedEntry struct {
	Index             int      `json:"index"`                        // Öğenin toplu işlemdeki dizini (0'dan başlar)
	Reason            string   `json:"reason"`                       // Başarısızlık nedeni
	FailureType       string   `json:"failure_type,omitempty"`       // Başarısızlık türü: pre_validation / post_validation
	IsPartialFailure  bool     `json:"is_partial_failure,omitempty"` // Kısmi başarısızlık olup olmadığı (benzer sorular/karşı örnekler kaldırıldı, ancak kaydın tamamı yine de içe aktarılabilir)
	TagName           string   `json:"tag_name,omitempty"`           // Kategori
	StandardQuestion  string   `json:"standard_question"`            // Standart soru
	SimilarQuestions  []string `json:"similar_questions,omitempty"`  // Benzer soru
	NegativeQuestions []string `json:"negative_questions,omitempty"` // Karşı örnek soru
	Answers           []string `json:"answers,omitempty"`            // Yanıt
	AnswerAll         bool     `json:"answer_all,omitempty"`         // Tümüne yanıt verilip verilmediği
	IsDisabled        bool     `json:"is_disabled,omitempty"`        // Devre dışı bırakılıp bırakılmadığı
	// Kısmi başarısızlık ayrıntıları (IsPartialFailure true olduğunda)
	RemovedSimilarQuestions  []string `json:"removed_similar_questions,omitempty"`  // Kaldırılan benzer sorular ve nedenleri
	RemovedNegativeQuestions []string `json:"removed_negative_questions,omitempty"` // Kaldırılan karşı örnekler ve nedenleri
}

// FAQMergeDetail, append modunda bir FAQ'ın mevcut chunk ile birleştirilmesinin sonuç özetini temsil eder.
type FAQMergeDetail struct {
	Index            int    `json:"index"`              // Satır numarası (parti içindeki indeks)
	StandardQuestion string `json:"standard_question"`  // Standart soru
	AnswerChanged    bool   `json:"answer_changed"`     // Yanıtın değiştirilip değiştirilmediği
	NewSimilarCount  int    `json:"new_similar_count"`  // Eklenen benzer soru sayısı
	NewNegativeCount int    `json:"new_negative_count"` // Eklenen karşı örnek sayısı
}

// FAQSuccessEntry, içe aktarma işlemi başarılı olan girdinin basit bilgilerini temsil eder
type FAQSuccessEntry struct {
	Index            int    `json:"index"`              // Girdinin parti içindeki indeksi (0'dan başlar)
	SeqID            int64  `json:"seq_id"`             // İçe aktarma sonrası girdi sıra ID'si
	TagID            int64  `json:"tag_id,omitempty"`   // Kategori ID'si (seq_id)
	TagName          string `json:"tag_name,omitempty"` // Kategori adı
	StandardQuestion string `json:"standard_question"`  // Standart soru
}

// FAQDryRunResult, dry_run modunun doğrulama sonucunu temsil eder
type FAQDryRunResult struct {
	TaskID        string           `json:"task_id,omitempty"` // Asenkron görev ID'si (asenkron modda döndürülür)
	Total         int              `json:"total"`             // Toplam kayıt sayısı
	SuccessCount  int              `json:"success_count"`     // Doğrulamayı geçen kayıt sayısı
	FailedCount   int              `json:"failed_count"`      // Doğrulaması başarısız olan kayıt sayısı
	FailedEntries []FAQFailedEntry `json:"failed_entries"`    // Başarısız kayıt ayrıntıları
}

// FAQSearchRequest FAQ arama istek parametreleri
type FAQSearchRequest struct {
	QueryText            string  `json:"query_text"             binding:"required"`
	VectorThreshold      float64 `json:"vector_threshold"`
	MatchCount           int     `json:"match_count"`
	FirstPriorityTagIDs  []int64 `json:"first_priority_tag_ids"`  // Birinci öncelikli etiket ID listesi; eşleşme kapsamını sınırlar, en yüksek önceliğe sahiptir
	SecondPriorityTagIDs []int64 `json:"second_priority_tag_ids"` // İkinci öncelikli etiket ID listesi; eşleşme kapsamını sınırlar, önceliği birinci öncelikten düşüktür
	OnlyRecommended      bool    `json:"only_recommended"`        // Yalnızca önerilen kayıtların döndürülüp döndürülmeyeceği
}

// UntaggedTagName is the default tag name for entries without a tag
const UntaggedTagName = "未分类"

// FAQEntryFieldsUpdate Tek bir FAQ kaydının alan güncellemesi
type FAQEntryFieldsUpdate struct {
	IsEnabled     *bool  `json:"is_enabled,omitempty"`
	IsRecommended *bool  `json:"is_recommended,omitempty"`
	TagID         *int64 `json:"tag_id,omitempty"`
	// Daha sonra daha fazla alan genişletilebilir
}

// FAQEntryFieldsBatchUpdate FAQ kayıt alanlarını toplu güncelleme isteği
// İki mod desteklenir:
// 1. Kayıt ID'sine göre güncelleme: ByID alanını kullanın
// 2. Etikete göre güncelleme: ByTag alanını kullanın; bu etiket altındaki tüm kayıtlara aynı güncellemeyi uygulayın
type FAQEntryFieldsBatchUpdate struct {
	// ByID kayıt ID'sine göre güncelleme; anahtar kayıt ID'sidir (seq_id)
	ByID map[int64]FAQEntryFieldsUpdate `json:"by_id,omitempty"`
	// ByTag etikete göre toplu güncelleme; anahtar TagID'dir (seq_id)
	ByTag map[int64]FAQEntryFieldsUpdate `json:"by_tag,omitempty"`
	// ExcludeIDs ByTag işleminde hariç tutulması gereken ID listesi (seq_id)
	ExcludeIDs []int64 `json:"exclude_ids,omitempty"`
}

// FAQImportTaskStatus içe aktarma görevi durumu
type FAQImportTaskStatus string

const (
	// FAQImportStatusPending represents the pending status of the FAQ import task
	FAQImportStatusPending FAQImportTaskStatus = "pending"
	// FAQImportStatusProcessing represents the processing status of the FAQ import task
	FAQImportStatusProcessing FAQImportTaskStatus = "processing"
	// FAQImportStatusCompleted represents the completed status of the FAQ import task
	FAQImportStatusCompleted FAQImportTaskStatus = "completed"
	// FAQImportStatusFailed represents the failed status of the FAQ import task
	FAQImportStatusFailed FAQImportTaskStatus = "failed"
)

// FAQImportProgress represents the progress of an FAQ import task stored in Redis
// When Status is "completed", the result fields (SkippedCount, ImportMode, ImportedAt, DisplayStatus, ProcessingTime) are populated.
type FAQImportProgress struct {
	TaskID             string              `json:"task_id"`                        // UUID for the import task
	KBID               string              `json:"kb_id"`                          // Knowledge Base ID
	KnowledgeID        string              `json:"knowledge_id"`                   // FAQ Knowledge ID
	Status             FAQImportTaskStatus `json:"status"`                         // Task status
	Progress           int                 `json:"progress"`                       // 0-100 percentage
	Total              int                 `json:"total"`                          // Total entries to import
	Processed          int                 `json:"processed"`                      // Entries processed so far
	SuccessCount       int                 `json:"success_count"`                  // Tamamen başarılı kayıt sayısı (kısmi başarılı/kısmi başarısız olanlar hariç)
	FailedCount        int                 `json:"failed_count"`                   // Başarısız kayıt sayısı
	PartialFailedCount int                 `json:"partial_failed_count,omitempty"` // Kısmen başarısız kayıt sayısı (benzer sorular/karşı örnekler kaldırıldı)
	SkippedCount       int                 `json:"skipped_count,omitempty"`        // Atlanan kayıt sayısı (ör. yinelenenler)
	FailedEntries      []FAQFailedEntry    `json:"failed_entries,omitempty"`       // Başarısız kayıt ayrıntıları (az sayıda olduğunda doğrudan döndürülür)
	FailedEntriesURL   string              `json:"failed_entries_url,omitempty"`   // Başarısız kayıt CSV indirme URL'si (çok sayıda olduğunda URL döndürülür)
	SuccessEntries     []FAQSuccessEntry   `json:"success_entries,omitempty"`      // Başarılı kayıtların kısa bilgileri (az sayıda olduğunda doğrudan döndürülür)
	ValidEntryIndices  []int               `json:"valid_entry_indices,omitempty"`  // Doğrulamayı geçen öğe dizinleri (yeniden denemede doğrulamayı atlamak için)
	MergeEntryIndices  []int               `json:"merge_entry_indices,omitempty"`  // Birleştirilecek öğe dizinleri (dahili kullanım; yeniden denemede tanımayı atlamak için)
	MergedCount        int                 `json:"merged_count,omitempty"`         // Birleştirilerek güncellenen öğe sayısı
	AddedCount         int                 `json:"added_count,omitempty"`          // Yeni eklenen öğe sayısı
	MergeDetails       []FAQMergeDetail    `json:"merge_details,omitempty"`        // Birleştirme ayrıntıları
	Message            string              `json:"message"`                        // Status message
	Error              string              `json:"error"`                          // Error message if failed
	CreatedAt          int64               `json:"created_at"`                     // Task creation timestamp
	UpdatedAt          int64               `json:"updated_at"`                     // Last update timestamp
	DryRun             bool                `json:"dry_run,omitempty"`              // dry run modunda olup olmadığı

	// Result fields (populated when Status == "completed")
	ImportMode     string    `json:"import_mode,omitempty"`     // İçe aktarma modu: append veya replace
	ImportedAt     time.Time `json:"imported_at,omitempty"`     // İçe aktarmanın tamamlanma zamanı
	DisplayStatus  string    `json:"display_status,omitempty"`  // Görüntüleme durumu: open veya close
	ProcessingTime int64     `json:"processing_time,omitempty"` // İşleme süresi (milisaniye)
}

// FAQImportMetadata, Knowledge.Metadata içinde FAQ içe aktarma görevi bilgilerini saklar
// Deprecated: Use FAQImportProgress with Redis storage instead
type FAQImportMetadata struct {
	ImportProgress  int `json:"import_progress"` // 0-100
	ImportTotal     int `json:"import_total"`
	ImportProcessed int `json:"import_processed"`
}

// FAQImportResult, FAQ içe aktarma tamamlandıktan sonraki istatistik sonuçlarını saklar
// Bu bilgi kalıcıdır; ilerleme durumunu takip etmez ve bir sonraki içe aktarmada değiştirilene kadar kalır
type FAQImportResult struct {
	// İçe aktarma istatistikleri
	TotalEntries       int `json:"total_entries"`        // Toplam öğe sayısı
	SuccessCount       int `json:"success_count"`        // Tamamen başarılı öğe sayısı (kısmi başarı/kısmi başarısızlık dahil değildir)
	FailedCount        int `json:"failed_count"`         // Tamamen başarısız öğe sayısı
	PartialFailedCount int `json:"partial_failed_count"` // Kısmen başarısız öğe sayısı (benzer sorular/karşı örnekler kaldırıldı ancak içe aktarıldı)
	SkippedCount       int `json:"skipped_count"`        // Atlanan öğe sayısı (ör. yinelenenler)
	MergedCount        int `json:"merged_count"`         // Birleştirilerek güncellenen öğe sayısı
	AddedCount         int `json:"added_count"`          // Yeni eklenen öğe sayısı

	// İçe aktarma modu ve zaman bilgileri
	ImportMode string    `json:"import_mode"` // İçe aktarma modu: append veya replace
	ImportedAt time.Time `json:"imported_at"` // İçe aktarmanın tamamlanma zamanı
	TaskID     string    `json:"task_id"`     // İçe aktarma görevi ID'si

	// Başarısızlık ayrıntıları URL'si (başarısız öğe sayısı fazla olduğunda indirme bağlantısı sağlanır)
	FailedEntriesURL string `json:"failed_entries_url,omitempty"` // Başarısız öğeler için CSV indirme URL'si

	// Görüntüleme denetimi
	DisplayStatus string `json:"display_status"` // Görüntüleme durumu: open veya close

	// Ek istatistiksel bilgiler
	ProcessingTime int64 `json:"processing_time"` // İşlem süresi (milisaniye)
}

// ToJSON converts the metadata to JSON type.
func (m *FAQImportMetadata) ToJSON() (JSON, error) {
	if m == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ToJSON converts the import result to JSON type.
func (r *FAQImportResult) ToJSON() (JSON, error) {
	if r == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ParseFAQImportMetadata parses FAQ import metadata from Knowledge.
func ParseFAQImportMetadata(k *Knowledge) (*FAQImportMetadata, error) {
	if k == nil || len(k.Metadata) == 0 {
		return nil, nil
	}
	var metadata FAQImportMetadata
	if err := json.Unmarshal(k.Metadata, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// normalizeQuestionStrings, soru listesini normalize eder
// Tam genişlikli karakterleri yarım genişlikli karakterlere dönüştürme, sondaki noktalama işaretlerini kaldırma, boşlukları birleştirme vb. işlemleri içerir; ayrıca yinelenenleri kaldırır
func normalizeQuestionStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	dedup := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		normalized := NormalizeQuestion(v)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		dedup = append(dedup, normalized)
	}
	if len(dedup) == 0 {
		return nil
	}
	return dedup
}

// multiSpaceRegex, birden fazla ardışık boşluk karakterini eşleştirmek için kullanılır
var multiSpaceRegex = regexp.MustCompile(`\s+`)

// urlRegex, URL eşleştirmek için kullanılır
var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// URLNormMode, URL normalleştirme modunu tanımlar
type URLNormMode int

const (
	// URLRemove, URL'yi tamamen kaldırır
	URLRemove URLNormMode = iota
	// URLPlaceholder, <URL> yer tutucusuyla değiştirir
	URLPlaceholder
	// URLKeepDomain, yalnızca alan adını korur
	URLKeepDomain
	// URLKeepDomainAndPath, alan adını ve yolu korur
	URLKeepDomainAndPath
)

// NormalizeQuestion, vektör eşleştirme isabet oranını artırmak için soru metnini normalize eder
// İşlem sırası referansı: query = convert_st(trim_url(query.lower().strip().strip("？。，；、：""！?.,;!:'\"")), 1)
// 1. Baş ve sondaki boşlukları kaldır
// 2. URL'yi kaldır
// 3. Küçük harfe dönüştür
// 4. Baş ve sondaki noktalama işaretlerini kaldır
// 5. Geleneksel yazıyı basitleştirilmiş yazıya dönüştür
// 6. Tam genişlikli sembolleri yarım genişlikli sembollere dönüştür
// 7. Akıllı boşluk işleme (karakterler arasındaki boşlukları kaldırır, Latin harfleri/sayılar arasındakileri korur)
func NormalizeQuestion(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}

	// 1. URL'yi kaldır
	q = trimURL(q)

	// 2. Küçük harfe dönüştür (İngilizce için geçerli)
	q = strings.ToLower(q)

	// 3. Baş ve sondaki noktalama işaretlerini kaldır
	q = strings.Trim(q, `？。，；、：""！?.,;!:'""`)

	// 4. Geleneksel karakterleri basitleştirilmiş karakterlere dönüştür
	q = toSimplified(q)

	// 5. Tam genişlikli karakterleri yarım genişlikli karakterlere dönüştür
	q = toHalfWidth(q)

	// 6. Akıllı boşluk işleme: Çince karakterler arasındaki boşlukları kaldır, İngilizce/rakamlar arasındakileri koru
	q = normalizeSpaces(q)

	return strings.TrimSpace(q)
}

// normalizeSpaces boşlukları akıllıca işler
// Kurallar:
// - Çince bağlamdaki gereksiz boşlukları kaldır
// - İngilizce/rakamlar arasındaki gerekli boşlukları koru
// Örnekler:
// - "Nasıl telefon bağlanır" → "Nasıltelefonbağlanır"
// - "iphone 15 nasıl etkinleştirilir" → "iphone 15nasıl etkinleştirilir"
func normalizeSpaces(s string) string {
	// Önce birden çok ardışık boşluğu tek bir boşlukta birleştir
	s = multiSpaceRegex.ReplaceAllString(s, " ")

	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(s))

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Boşluk değilse doğrudan yaz
		if r != ' ' {
			builder.WriteRune(r)
			continue
		}

		// Boşluğu işle: korunup korunmayacağına karar vermek için önceki ve sonraki karakterleri kontrol et
		// Önceki boşluk olmayan karakteri al
		var prevRune rune
		if i > 0 {
			prevRune = runes[i-1]
		}

		// Sonraki boşluk olmayan karakteri al
		var nextRune rune
		for j := i + 1; j < len(runes); j++ {
			if runes[j] != ' ' {
				nextRune = runes[j]
				break
			}
		}

		// Boşluğu yalnızca öncesi ve sonrası İngilizce harf veya rakam olduğunda koru
		// Diğer tüm durumlarda (Çince dâhil) boşluğu kaldır
		if isASCIIAlphaNum(prevRune) && isASCIIAlphaNum(nextRune) {
			builder.WriteRune(' ')
		}
		// Aksi hâlde boşluğu atla
	}

	return builder.String()
}

// isASCIIAlphaNum ASCII harfi veya rakamı olup olmadığını kontrol eder
func isASCIIAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// trimURL dizedeki URL'leri kaldırır (varsayılan URLRemove modu kullanılarak)
func trimURL(s string) string {
	return NormalizeURL(s, URLRemove)
}

// NormalizeURL metindeki URL'leri belirtilen moda göre işler
func NormalizeURL(text string, mode URLNormMode) string {
	return urlRegex.ReplaceAllStringFunc(text, func(raw string) string {
		switch mode {
		case URLRemove:
			return ""
		case URLPlaceholder:
			return "<URL>"
		case URLKeepDomain:
			domain, _ := parseURL(raw)
			if domain != "" {
				return domain
			}
			return "<URL>"
		case URLKeepDomainAndPath:
			domain, path := parseURL(raw)
			if domain != "" {
				return domain + path
			}
			return "<URL>"
		default:
			return "<URL>"
		}
	})
}

// parseURL URL'yi ayrıştırır, alan adını ve yolu döndürür
func parseURL(raw string) (domain, path string) {
	// Protokol önekini kaldır
	u := raw
	if strings.HasPrefix(u, "https://") {
		u = u[8:]
	} else if strings.HasPrefix(u, "http://") {
		u = u[7:]
	}

	// Alan adı ve yolu ayır
	slashIdx := strings.Index(u, "/")
	if slashIdx == -1 {
		// Yol yok, tamamı alan adıdır (sorgu parametreleri içerebilir)
		queryIdx := strings.Index(u, "?")
		if queryIdx != -1 {
			domain = u[:queryIdx]
		} else {
			domain = u
		}
		return domain, ""
	}

	domain = u[:slashIdx]
	path = u[slashIdx:]

	// Sorgu parametrelerini ve parçayı kaldır
	if queryIdx := strings.Index(path, "?"); queryIdx != -1 {
		path = path[:queryIdx]
	}
	if fragIdx := strings.Index(path, "#"); fragIdx != -1 {
		path = path[:fragIdx]
	}

	return domain, path
}

// toSimplified Geleneksel Çinceyi Basitleştirilmiş Çinceye dönüştür
func toSimplified(s string) string {
	return textconv.ToSimplified(s)
}

// toHalfWidth Tam genişlikli karakterleri yarım genişlikli karakterlere dönüştür
// Başlıca işlenenler: tam genişlikli boşluklar, tam genişlikli ASCII karakterleri (noktalama işaretleri dahil)
func toHalfWidth(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	for _, r := range s {
		switch {
		// Tam genişlikli boşluk -> yarım genişlikli boşluk
		case r == '\u3000':
			builder.WriteRune(' ')
		// Tam genişlikli ASCII karakterleri (！ ile ～ arası, aralık 0xFF01-0xFF5E) -> yarım genişlikli (0x0021-0x007E)
		case r >= 0xFF01 && r <= 0xFF5E:
			builder.WriteRune(r - 0xFF01 + 0x21)
		// Diğer karakterler değişmeden kalır
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// NormalizeQueryText arama sorgusu metnini normalleştirir
// NormalizeQuestion ile aynı işleme mantığı; arama sırasında sorgu metnini normalleştirmek için kullanılır
func NormalizeQueryText(q string) string {
	return NormalizeQuestion(q)
}

// IsChineseChar bir karakterin Çince olup olmadığını belirler
func IsChineseChar(r rune) bool {
	return unicode.Is(unicode.Han, r)
}
