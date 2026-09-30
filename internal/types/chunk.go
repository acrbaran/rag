// Package types defines data structures and types used throughout the system
// These types are shared across different service modules to ensure data consistency
package types

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// ChunkType, farklı Chunk türlerini tanımlar
type ChunkType = string

const (
	// ChunkTypeText, normal metin Chunk'ını temsil eder
	ChunkTypeText ChunkType = "text"
	// ChunkTypeParentText, üst-alt parçalama stratejisindeki üst metin Chunk'ını temsil eder (yalnızca bağlam için kullanılır, vektör indekslemeye katılmaz)
	ChunkTypeParentText ChunkType = "parent_text"
	// ChunkTypeImageOCR, görüntü OCR metninin Chunk'ını temsil eder
	ChunkTypeImageOCR ChunkType = "image_ocr"
	// ChunkTypeImageCaption, görüntü açıklamasının Chunk'ını temsil eder
	ChunkTypeImageCaption ChunkType = "image_caption"
	// ChunkTypeSummary, özet türündeki Chunk'ı temsil eder
	ChunkTypeSummary = "summary"
	// ChunkTypeEntity, varlık türündeki Chunk'ı temsil eder
	ChunkTypeEntity ChunkType = "entity"
	// ChunkTypeRelationship, ilişki türündeki Chunk'ı temsil eder
	ChunkTypeRelationship ChunkType = "relationship"
	// ChunkTypeFAQ, SSS öğesi Chunk'ını temsil eder
	ChunkTypeFAQ ChunkType = "faq"
	// ChunkTypeWebSearch, Web arama sonuçlarının Chunk'ını temsil eder
	ChunkTypeWebSearch ChunkType = "web_search"
	// ChunkTypeTableSummary, veri tablosu özetinin Chunk'ını temsil eder
	ChunkTypeTableSummary ChunkType = "table_summary"
	// ChunkTypeTableColumn, veri tablosu sütun açıklamasının Chunk'ını temsil eder
	ChunkTypeTableColumn ChunkType = "table_column"
	// ChunkTypeWikiPage, wiki sayfası senkronizasyonunun Chunk'ını temsil eder; wiki sayfalarını mevcut arama hattına dahil etmek için kullanılır
	ChunkTypeWikiPage ChunkType = "wiki_page"
)

// ChunkStatus, farklı Chunk durumlarını tanımlar
type ChunkStatus int

const (
	ChunkStatusDefault ChunkStatus = 0
	// ChunkStatusStored, depolanmış Chunk'ı temsil eder
	ChunkStatusStored ChunkStatus = 1
	// ChunkStatusIndexed, indekslenmiş Chunk'ı temsil eder
	ChunkStatusIndexed ChunkStatus = 2
)

// ChunkFlags, birden fazla boole durumunu yönetmek için Chunk'ın bayrak bitlerini tanımlar
type ChunkFlags int

const (
	// ChunkFlagRecommended, önerilebilir durumu temsil eder（1 << 0 = 1）
	// Bu bayrak ayarlandığında, ilgili Chunk kullanıcıya önerilebilir
	ChunkFlagRecommended ChunkFlags = 1 << 0
	// Gelecekte daha fazla bayrak biti eklenebilir:
	// ChunkFlagPinned ChunkFlags = 1 << 1  // Sabitlenmiş
	// ChunkFlagHot    ChunkFlags = 1 << 2  // Popüler
)

// HasFlag, belirtilen bayrağın ayarlanıp ayarlanmadığını kontrol eder
func (f ChunkFlags) HasFlag(flag ChunkFlags) bool {
	return f&flag != 0
}

// SetFlag, belirtilen bayrağı ayarlar
func (f ChunkFlags) SetFlag(flag ChunkFlags) ChunkFlags {
	return f | flag
}

// ClearFlag, belirtilen bayrağı temizler
func (f ChunkFlags) ClearFlag(flag ChunkFlags) ChunkFlags {
	return f &^ flag
}

// ToggleFlag, belirtilen bayrağı değiştirir
func (f ChunkFlags) ToggleFlag(flag ChunkFlags) ChunkFlags {
	return f ^ flag
}

// ImageInfo, Chunk ile ilişkili görüntü bilgilerini temsil eder
type ImageInfo struct {
	// SHA256 identifies the exact stored image bytes across parser and preview.
	SHA256 string `json:"sha256,omitempty"`
	// Görüntü URL'si（COS）
	URL string `json:"url"          gorm:"type:text"`
	// Orijinal görüntü URL'si
	OriginalURL string `json:"original_url" gorm:"type:text"`
	// Görüntünün metindeki başlangıç konumu
	StartPos int `json:"start_pos"`
	// Görüntünün metindeki bitiş konumu
	EndPos int `json:"end_pos"`
	// Görüntü açıklaması
	Caption string `json:"caption"`
	// Görüntü OCR metni
	OCRText string `json:"ocr_text"`
	// Attrs, model turunun verdiği görüntü özniteliği gözlem sonuçlarını açıklar (değerler için ImageAttrs / ImageAttrRegistry'ye bakın).
	// Yalnızca modelin gerçekten yanıtladığı öznitelikler kaydedilir: gözlemi başarısız olan veya değeri geçersiz olan özniteliklere varsayılan değer yazılmaz, doğrudan anahtarları eksik bırakılır,
	// bu nedenle okuyucu, 「gözlemlenmedi」 ile 「negatif değer gözlemlendi」 durumlarını ImageAttrs.Observed kullanarak ayırt etmelidir.
	// Bu alan chunks.image_info içinde JSON olarak saklanır, geçiş gerektirmez; öznitelik gözlem yeteneği kullanıma alınmadan önce yazılan
	// satırlar varsayılan olarak boş olacaktır; okuyucu boş değerleri tolere etmelidir.
	Attrs ImageAttrs `json:"attrs,omitempty"`
}

// VideoInfo, Chunk ile ilişkili video bilgilerini temsil eder
type VideoInfo struct {
	// Video URL'si
	URL string `json:"url"          gorm:"type:text"`
}

// Chunk represents a document chunk
// Chunks are meaningful text segments extracted from original documents
// and are the basic units of knowledge base retrieval
// Each chunk contains a portion of the original content
// and maintains its positional relationship with the original text
// Chunks can be independently embedded as vectors and retrieved, supporting precise content localization
type Chunk struct {
	// Unique identifier of the chunk, using UUID format
	ID string `json:"id"                       gorm:"type:varchar(36);primaryKey"`
	// SeqID is an auto-increment integer ID for external API usage (FAQ entries)
	SeqID int64 `json:"seq_id"                   gorm:"type:bigint;uniqueIndex;autoIncrement"`
	// Tenant ID, used for multi-tenant isolation
	TenantID uint64 `json:"tenant_id"`
	// ID of the parent knowledge, associated with the Knowledge model
	KnowledgeID string `json:"knowledge_id"`
	// ID of the knowledge base, for quick location
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// Optional tag ID for categorization within a knowledge base (used for FAQ)
	TagID string `json:"tag_id"                   gorm:"type:varchar(36);index"`
	// Actual text content of the chunk
	Content string `json:"content"`
	// SourceContent is the immutable parser output. Legacy rows are lazily
	// backfilled from Content on the first manual edit.
	SourceContent string `json:"-"`
	// ContentRevision is incremented for every user edit or rollback.
	ContentRevision int `json:"content_revision" gorm:"not null;default:0"`
	// IndexStatus reports whether the current content is reflected in the
	// retrieval stores: ready | processing | failed.
	IndexStatus string `json:"index_status" gorm:"type:varchar(16);not null;default:'ready'"`
	// LastEditorID records the actor that produced the current revision.
	LastEditorID string `json:"last_editor_id" gorm:"type:varchar(64);not null;default:''"`
	// Index position of the chunk in the original document
	ChunkIndex int `json:"chunk_index"`
	// Whether the chunk is enabled, can be used to temporarily disable certain chunks
	IsEnabled bool `json:"is_enabled"               gorm:"default:true"`
	// Flags, birden fazla boole durumunun bit bayraklarını saklar (öneri durumu vb.)
	// Varsayılan değer ChunkFlagRecommended (1)'dir ve varsayılan olarak önerilebilir olduğunu belirtir
	Flags ChunkFlags `json:"flags"                    gorm:"default:1"`
	// Status of the chunk
	Status int `json:"status"                   gorm:"default:0"`
	// Starting character position in the original text
	StartAt int `json:"start_at"`
	// Ending character position in the original text
	EndAt int `json:"end_at"`
	// Previous chunk ID
	PreChunkID string `json:"pre_chunk_id"`
	// Next chunk ID
	NextChunkID string `json:"next_chunk_id"`
	// Farklı Chunk türlerini ayırt etmek için kullanılan Chunk türü
	ChunkType ChunkType `json:"chunk_type"               gorm:"type:varchar(20);default:'text'"`
	// Görüntü Chunk'ını özgün metin Chunk'ıyla ilişkilendirmek için üst Chunk ID'si
	ParentChunkID string `json:"parent_chunk_id"          gorm:"type:varchar(36);index"`
	// İlişki Chunk'ını özgün metin Chunk'ıyla ilişkilendirmek için ilişki Chunk ID'si
	RelationChunks JSON `json:"relation_chunks"          gorm:"type:json"`
	// Dolaylı ilişki Chunk'ını özgün metin Chunk'ıyla ilişkilendirmek için dolaylı ilişki Chunk ID'si
	IndirectRelationChunks JSON `json:"indirect_relation_chunks" gorm:"type:json"`
	// Metadata, FAQ meta verileri gibi chunk düzeyindeki ek bilgileri saklar
	Metadata JSON `json:"metadata"                 gorm:"type:json"`
	// ContentHash, hızlı eşleştirme için içeriğin hash değerini saklar (esas olarak FAQ için)
	ContentHash string `json:"content_hash"             gorm:"type:varchar(64)"`
	// JSON olarak saklanan görüntü bilgisi
	ImageInfo string `json:"image_info"               gorm:"type:text"`
	// Chunk creation time
	CreatedAt time.Time `json:"created_at"`
	// Chunk last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete marker, supports data recovery
	DeletedAt gorm.DeletedAt `json:"deleted_at"               gorm:"index"`
	// ContextHeader is a Markdown heading breadcrumb prepended when indexing.
	// It is persisted so a later content edit can rebuild the same index input.
	ContextHeader string `json:"-" gorm:"type:text"`
	// SourceLocators point back into the original file (page and region,
	// slide, sheet rows, ...) so citations can open the file at this chunk.
	// Empty when the parser reported no positions.
	SourceLocators SourceLocators `json:"source_locators,omitempty" gorm:"type:json"`
}

// ChunkRevision is an immutable snapshot of a superseded chunk revision.
// The current content lives on Chunk; this table stores prior versions.
type ChunkRevision struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id" gorm:"index"`
	KnowledgeBaseID string    `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	KnowledgeID     string    `json:"knowledge_id" gorm:"type:varchar(36);index"`
	ChunkID         string    `json:"chunk_id" gorm:"type:varchar(36);uniqueIndex:idx_chunk_revision"`
	Revision        int       `json:"revision" gorm:"uniqueIndex:idx_chunk_revision"`
	Content         string    `json:"content" gorm:"type:text"`
	IsEnabled       bool      `json:"is_enabled"`
	EditorID        string    `json:"editor_id" gorm:"type:varchar(64)"`
	EditSource      string    `json:"edit_source" gorm:"type:varchar(16)"`
	EditedAt        time.Time `json:"edited_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// EmbeddingContent returns the chunk content with ContextHeader prepended
// when set. Use this where the embedding model needs section context that
// isn't part of the literal Content. Surrounding whitespace on Content is
// trimmed so leading/trailing newlines from boundary slicing don't dilute
// the embedded vector.
func (c *Chunk) EmbeddingContent() string {
	if c == nil {
		return ""
	}
	body := strings.TrimSpace(c.Content)
	if c.ContextHeader == "" {
		return body
	}
	return c.ContextHeader + "\n\n" + body
}

// AssignChunkSeqIDs assigns sequential SeqIDs to a batch of chunks that have SeqID == 0.
// Must be called before CreateInBatches for SQLite compatibility.
func AssignChunkSeqIDs(tx *gorm.DB, chunks []*Chunk) error {
	needAssign := false
	for _, c := range chunks {
		if c.SeqID == 0 {
			needAssign = true
			break
		}
	}
	if !needAssign {
		return nil
	}

	var maxSeqID *int64
	if err := tx.Unscoped().Model(&Chunk{}).Select("MAX(seq_id)").Scan(&maxSeqID).Error; err != nil {
		return err
	}
	next := int64(1)
	if maxSeqID != nil {
		next = *maxSeqID + 1
	}
	for _, c := range chunks {
		if c.SeqID == 0 {
			c.SeqID = next
			next++
		}
	}
	return nil
}
