package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/handler"
)

// RegisterChunkRoutes, parça ile ilgili rotaları kaydeder
//
// Mutating routes addressed via :knowledge_id inherit per-KB ownership
// from the owning knowledge entry's KB (PR 5, #1303); the chain hop is
// shared with RegisterKnowledgeRoutes via OwnedChunkKBOrAdmin so the
// same "creator-of-the-KB OR Admin+" rule applies to chunk edits.
func RegisterChunkRoutes(r *gin.RouterGroup, handler *handler.ChunkHandler, g *rbacGuards) {
	// Parça rota grubu. Scoped API key, içerik yazmak için ingest, içerik okumak için retrieve yeteneğine ihtiyaç duyar;
	// Her ikisi de hâlâ KB beyaz listesi kısıtlamasına tabidir。
	chunks := g.apiKeyGroup(r.Group("/chunks"), apiKeyIngest(apiKeyFullAccess()))
	chunkRead := chunks.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Parça listesini al — Viewer+ ve üst KB için read izni (own / shared / via shared agent)
		chunkRead.GET("/:knowledge_id", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("knowledge_id"), handler.ListKnowledgeChunks)
		// chunk_id ile tek bir chunk al (knowledge_id gerekmez) — Viewer+ ve üst KB için read izni
		chunkRead.GET("/by-id/:id", g.Viewer(), g.KBAccessReadFromChunkIDParam("id"), handler.GetChunkByIDOnly)
		chunkRead.GET("/:knowledge_id/:id/revisions", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("knowledge_id"), handler.ListChunkRevisions)
		// Parçayı sil — KB owner VEYA Admin+ ve üst KB için write izni
		chunks.DELETE("/:knowledge_id/:id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.DeleteChunk)
		// Bilgi altındaki tüm parçaları sil — KB owner VEYA Admin+ ve üst KB için write izni
		chunks.DELETE("/:knowledge_id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.DeleteChunksByKnowledgeID)
		// Parça bilgilerini güncelle — KB owner VEYA Admin+ ve üst KB için write izni
		chunks.PUT("/:knowledge_id/:id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.UpdateChunk)
		chunks.POST("/:knowledge_id/:id/revert", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.RevertChunk)
		// Tek bir oluşturulmuş soruyu sil (parça id üzerinden) — diğer chunk mutation işlemleriyle tutarlı:
		// KB owner VEYA Admin+。Başlangıçta burada zincir (chunk_id -> knowledge_id ->
		// kb -> creator_id) henüz bağlı olmadığı için geçici olarak Contributor seviyesine düşürüldü; bu da
		// 「tüm chunk'ları düzenleyebilme için aynı kuralın bu rotada daha gevşek olması」tutarsızlığına yol açtı。
		// Şimdi KBCreatorLookupFromChunkIDParam ile bu atlama eklenerek matris birleştirildi。
		chunks.DELETE("/by-id/:id/questions", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.DeleteGeneratedQuestion)
		chunks.PUT("/by-id/:id/questions", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.UpsertGeneratedQuestion)
		chunks.POST("/by-id/:id/questions/regenerate", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.RegenerateGeneratedQuestions)
	}
}

// RegisterKnowledgeRoutes bilgiyle ilgili rotaları kaydeder
//
// Per-KB ownership applies on the per-:id mutating routes (PR 5,
// #1303): the URL :id is a knowledge id, OwnedKnowledgeKBOrAdmin
// walks it back to KB.CreatorID so a Contributor who owns the KB can
// edit/delete any of its documents while a non-owner Contributor gets
// 403. KB-scoped upload routes (`/knowledge-bases/:id/knowledge/...`)
// reuse OwnedKBOrAdmin because the URL :id is the KB id directly.
// Body-scoped batch operations have a Contributor route gate and resolve
// their KB ownership plus Editor operation grant inside the handler.
func RegisterKnowledgeRoutes(r *gin.RouterGroup, handler *handler.KnowledgeHandler, g *rbacGuards) {
	// Bilgi tabanı altındaki bilgi rota grubu (URL :id is the KB id)。Scoped API key için
	// içerik yazmak üzere ingest yeteneği gerekir ve KB kapsamı sınırına tabi olmaya devam eder; KB'yi temizleme yalnızca full-access key için izinlidir。
	kb := g.apiKeyGroup(r.Group("/knowledge-bases/:id/knowledge"), apiKeyIngest(apiKeyFullAccess()))
	kbRead := kb.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		kb.POST("/file", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateKnowledgeFromFile)
		kb.POST("/url", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateKnowledgeFromURL)
		kb.POST("/manual", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateManualKnowledge)
		kbRead.GET("", g.Viewer(), g.KBAccessRead("id"), handler.ListKnowledge)
		// Orijinal dosya indirme, tek dosya indirmenin Contributor + Editor yetki sınırını kullanmaya devam eder。
		kbRead.POST("/batch-download", g.Contributor(), g.KBAccessWrite("id"), handler.BatchDownloadKnowledge)
		kbRead.GET("/folders", g.Viewer(), g.KBAccessRead("id"), handler.ListKnowledgeFolders)
		kb.PUT("/folders", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.RenameKnowledgeFolder)
		// Clearing all contents under a KB is a destructive op; gate
		// behind Admin instead of Contributor.
		kb.With(apiKeyFullAccess()).DELETE("", g.Admin(), g.KBAccessWrite("id"), handler.ClearKnowledgeBaseContents)
	}

	// Image gallery: list every image asset of a KB (read-only, Viewer+).
	// Lives directly under /knowledge-bases/:id so it parallels /knowledge,
	// /faq and /tags rather than nesting under /knowledge (documents).
	kbImages := g.apiKeyGroup(r.Group("/knowledge-bases/:id"), apiKeyRetrieve(apiKeyFullAccess()))
	kbImagesRead := kbImages.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		kbImagesRead.GET("/images", g.Viewer(), g.KBAccessRead("id"), handler.ListImages)
		// Self-describing gallery contract: live attribute sources, the
		// resolved attribute list (definitions + merged usage) and the
		// caller's search activation state. Per-KB because KB-defined
		// attribute sources resolve against :id.
		kbImagesRead.GET("/gallery-config", g.Viewer(), g.KBAccessRead("id"), handler.GetGalleryConfig)
	}

	// Bilgi rota grubu (URL :id is a knowledge id; guard bunu üst KB'ye kadar takip eder)
	kgrp := r.Group("/knowledge")
	k := g.apiKeyGroup(kgrp, apiKeyIngest(apiKeyFullAccess()))
	kRead := k.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Cross-knowledge endpoints (no :id) can't be gated on a single
		// KB via the URL — they accept a kb_id (or source/target KB) in the
		// body and the handler fans out the access check itself. /batch and
		// /search are read routes (retrieve). /move, /batch-delete,
		// /batch-reparse and /tags are content writes that each bound
		// themselves to a single (or source+target) KB and enforce the API
		// key's KB allow-list downstream — MoveKnowledge via
		// requireTenantAPIKeyKnowledgeBases(source,target); the batch ops via
		// validateKnowledgeBaseAccessWithKBID (requireTenantAPIKeyKnowledgeBase)
		// plus a per-item "belongs to the authorized KB" check in the handler
		// and service. They are therefore declared for API keys under the
		// ingest capability, matching their single-document siblings
		// (k.DELETE("/:id"), k.POST("/:id/reparse"), k.PUT("/:id")).
		kRead.GET("/batch", g.Viewer(), handler.GetKnowledgeBatch)
		kRead.GET("/:id", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledge)
		kRead.GET("/:id/stages", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledgeSpans)
		kRead.GET("/:id/spans", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledgeSpans)
		k.DELETE("/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.DeleteKnowledge)
		k.PUT("/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateKnowledge)
		k.POST("/:id/regenerate-summary", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.RegenerateKnowledgeSummary)
		k.PUT("/manual/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateManualKnowledge)
		k.POST("/:id/reparse", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.ReparseKnowledge)
		k.POST("/:id/cancel-parse", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.CancelKnowledgeParse)
		// Downloading exposes the original source file, so it has a stricter
		// boundary than viewing parsed content or previewing it: tenant Viewers
		// cannot download from their own workspace, and org-shared Viewer access
		// cannot download from the source workspace. API keys still follow the
		// retrieve capability declared by kRead; role guards intentionally defer
		// machine-principal authorization to the API-key gate.
		kRead.GET("/:id/download", g.Contributor(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.DownloadKnowledgeFile)
		kRead.GET("/:id/preview", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.PreviewKnowledgeFile)
		k.PUT("/image/:id/:chunk_id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateImageInfo)
		kRead.GET("/search", g.Viewer(), handler.SearchKnowledge)
		kRead.GET("/move/progress/:task_id", g.Viewer(), handler.GetKnowledgeMoveProgress)
		// Batch / cross-KB content writes: JWT Contributor+, or an API key
		// with the ingest capability (or full access). Each handler binds the
		// operation to a single (move: source+target) KB and rejects any KB
		// or knowledge id outside the key's allow-list, so a scoped ingest key
		// can only touch KBs it is already permitted to write.
		k.PUT("/tags", g.Contributor(), handler.UpdateKnowledgeTagBatch)
		k.POST("/batch-reparse", g.Contributor(), handler.BatchReparseKnowledge)
		k.POST("/batch-delete", g.Contributor(), handler.BatchDeleteKnowledge)
		k.POST("/folder", g.Contributor(), handler.MoveKnowledgeToFolder)
		k.POST("/move", g.Contributor(), handler.MoveKnowledge)
	}
}

// RegisterFAQRoutes FAQ ile ilgili rotaları kaydeder
//
// FAQ entries are KB content: reads are Viewer+, all mutations
// (create / update / upsert / delete / batch field+tag updates,
// import display flag) are Contributor+. Search is read-only.
func RegisterFAQRoutes(r *gin.RouterGroup, handler *handler.FAQHandler, g *rbacGuards) {
	if handler == nil {
		return
	}
	// FAQ entries, KB'nin alt kaynaklarıdır (FAQ-type KB'nin ana içerik gövdesi)。FAQ'yi değiştirmek
	// KB içeriğini değiştirmeye eşdeğerdir ve KB'nin "creator OR Admin+" matrisine uymalıdır ——
	// chunks / wiki pages ile tutarlı olarak。Viewer+ okuyabilir, Contributor ise kendisine ait olmayan KB'nin FAQ'sini
	// değiştiremez。
	faq := g.apiKeyGroup(r.Group("/knowledge-bases/:id/faq"), apiKeyIngest(apiKeyFullAccess()))
	faqRead := faq.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// KBAccessRead/Write resolve own/shared/agent-visible access and
		// rewrite the request's tenant context — handler no longer
		// carries an effectiveCtxForKB helper.
		faqRead.GET("/entries", g.Viewer(), g.KBAccessRead("id"), handler.ListEntries)
		faqRead.GET("/entries/export", g.Viewer(), g.KBAccessRead("id"), handler.ExportEntries)
		faqRead.GET("/entries/:entry_id", g.Viewer(), g.KBAccessRead("id"), handler.GetEntry)
		faq.POST("/entries", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpsertEntries)
		faq.POST("/entry", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateEntry)
		faq.PUT("/entries/:entry_id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpdateEntry)
		faq.POST("/entries/:entry_id/similar-questions", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.AddSimilarQuestions)
		// Unified batch update API - supports is_enabled, is_recommended, tag_id
		faq.PUT("/entries/fields", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpdateEntryFieldsBatch)
		faq.PUT("/entries/tags", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpdateEntryTagBatch)
		faq.DELETE("/entries", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.DeleteEntries)
		// Search is a read route: scoped API keys may call it with retrieve
		// even though POST is otherwise an unsafe method.
		faqRead.POST("/search", g.Viewer(), g.KBAccessRead("id"), handler.SearchFAQ)
		// FAQ import result display status
		faq.PUT("/import/last-result/display", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpdateLastImportResultDisplayStatus)
	}
	// FAQ import progress route (outside of knowledge-base scope) — Viewer+.
	// Scoped API keys that can ingest (they start the import) or retrieve may
	// poll their own import/dry-run progress. The task is tenant-scoped by
	// requireTaskProgressTenant, so a key only ever sees its own tenant's
	// tasks. Declared through apiKeyRoute so the APIKeyGate doesn't fail-closed
	// and 403 the poller with "scope does not allow this operation".
	g.apiKeyRoute(r, http.MethodGet, "/faq/import/progress/:task_id",
		apiKeyRetrieve(apiKeyIngest(apiKeyFullAccess())), g.Viewer(), handler.GetImportProgress)
}

// RegisterKnowledgeBaseRoutes bilgi tabanıyla ilgili rotaları kaydeder
func RegisterKnowledgeBaseRoutes(r *gin.RouterGroup, handler *handler.KnowledgeBaseHandler, g *rbacGuards) {
	// Bilgi tabanı rota grubu。API-key erişilebilirliği yeteneklere göre iki seviyeye ayrılır; tümü apiKeyGroup üzerinden bildirilir,
	// artık çıplak kbgrp.Handle kaydı kullanmayın (bu, ağ geçidini atlar ve tüm key'ler için sessizce varsayılan reddetme uygular):
	//
	//   1. Okuma（list/detail/search/progress/move-targets）—— retrieve OR full-access（kb）
	//   2. KB yaşam döngüsü yönetimi（create/copy/duplicate/update/delete）
	//      —— manage_kbs OR full-access（kbManagement）
	//
	// Seviye 2'de tüm KB yaşam döngüsü aynı stratejiyi paylaşır: manage_kbs, 「bilgi tabanlarını yönetme」 capability'sidir,
	// oluşturma/kopyalama/güncelleme/silme onun sorumluluğundadır. KB allow-list'i alt katmanda geçerliliğini korur——copy/duplicate/
	// update/delete için hedef KB allow-list tarafından korunur; create için kısıtlanacak kaynak yoktur, allow-list ile sınırlandırılmış
	// bir key'in oluşturduğu yeni KB kendi allow-list'inin dışında kalır (aynı tenant, yetki aşımı yoktur; yalnızca oluşturduktan sonra kendisi yönetemez),
	// boş allow-list'e sahip key ise tenant genelinde KB yönetimidir; yeni oluşturulanlar doğal olarak kapsam içindedir. KB içeriği yazımı (belge/
	// parça/FAQ/Tag/Wiki) ilgili alt rotanın ingest yeteneği tarafından kontrol edilir, bu gruba dahil değildir.
	kbgrp := r.Group("/knowledge-bases")
	kb := g.apiKeyGroup(kbgrp, apiKeyRetrieve(apiKeyFullAccess()))
	kbManagement := kb.With(apiKeyManageKnowledgeBases(apiKeyFullAccess()))
	{
		// Bilgi tabanı oluşturma — JWT Contributor+; API key için manage_kbs veya full-access gerekir.
		kbManagement.POST("", g.Contributor(), handler.CreateKnowledgeBase)
		// Bilgi tabanı listesini alma — JWT çağıranlar için Viewer+; retrieve-capable API key'ler geçitten geçer.
		kb.GET("", g.Viewer(), handler.ListKnowledgeBases)
		// Bilgi tabanı detaylarını alma — Viewer+ ve KB için read izni
		kb.GET("/:id", g.Viewer(), g.KBAccessRead("id"), handler.GetKnowledgeBase)
		// Bilgi tabanını güncelleme/silme — birbirinden bağımsız iki katmanlı yetkilendirme, ikisi de zorunludur:
		//   OwnedKBOrAdmin  「tenant içi」sahipliği yönetir: Oluşturan olmayan Contributor,
		//                   iş arkadaşının KB'sini değiştiremez (tenant'lar arası KB burada lookup=NotFound →
		//                   olarak ilerler ve alt katmana bırakılır; burada engellenmez).
		//   KBAccessWrite   「tenant'lar arası」erişim düzeyini yönetir: Kendi KB'si veya kuruluş tarafından paylaşılan(editor).
		// handler içinde permission/sahip tenant'a göre nihai karar yeniden verilir —— özellikle DeleteKnowledgeBase
		// çağıranın 「kendi」tenant'ını(c.Keys, KBAccess tarafından yeniden yazılmamış) kullanarak kb.TenantID'yi doğrular,
		// silme işlemini 「sahip tenant + Admin」ile kilitler; paylaşılan editor kaynak KB'yi silemez.
		kbManagement.PUT("/:id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.UpdateKnowledgeBase)
		// Bilgi tabanının AI açıklamasını hemen yeniden oluşturma — bilgi tabanını güncelleme ile aynı seviye yetkilendirme; küçük model çağrısı bir kez eşzamanlı çalıştırılır.
		kbManagement.POST("/:id/profile/generate", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"),
			handler.GenerateKnowledgeBaseProfile)
		kbManagement.DELETE("/:id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.DeleteKnowledgeBase)
		// Bilgi tabanını sabitleme/sabitlemeyi kaldırma — bizzat oluşturan OR Admin+ ve KB için write izni
		// Pin state is now per-(user, kb) (migration 000050). Anyone with
		// at least Viewer-level read access to the KB — including users
		// who reached it via a shared agent — may pin it for themselves;
		// no edit permission is required. The OwnedKBOrAdmin guard was
		// removed accordingly. The route still requires KB read access
		// so callers can't poke at KBs they can't see.
		kb.PUT("/:id/pin", g.Viewer(), g.KBAccessRead("id"), handler.TogglePinKnowledgeBase)
		// Karma arama — Viewer+ ve KB için read izni (read-only)
		// POST is preferred; GET with JSON body is kept for backward compatibility (#1727).
		kb.POST("/:id/hybrid-search", g.Viewer(), g.KBAccessRead("id"), handler.HybridSearch)
		kb.GET("/:id/hybrid-search", g.Viewer(), g.KBAccessRead("id"), handler.HybridSearch)
		// Bilgi tabanını kopyalama — yeni bir KB üretir ve create ile aynı seviyededir: JWT Contributor+; API key için manage_kbs veya full-access gerekir.
		// Kaynak KB, body içindeki source_id ile iletilir (:id yol parametresi değil); yol parametresine dayalı
		// KBAccessRead uygulanamaz; bu nedenle kaynak/hedef KB'nin tenant sahipliği ve allow-list doğrulaması handler içinde tamamlanır
		// (requireTenantAPIKeyKnowledgeBases, source_id/target_id'yi allow-list içine alır).
		// Kopya çağırana aittir; özgün KB'nin sahipliği gerekmez.
		kbManagement.POST("/copy", g.Contributor(), handler.CopyKnowledgeBase)
		// Bilgi bankası kopyası oluştur — yeni bir KB üretir, create ile aynı düzeyde: JWT Contributor+; API anahtarı için manage_kbs veya full-access gerekir;
		// ve kaynak KB için read izni gerekir (KBAccessRead, kısıtlı anahtarlar için kaynak KB'yi kapsar). Yalnızca yeni KB ayar kaydı oluşturulur; içerik/indeks/paylaşımlar kopyalanmaz.
		kbManagement.POST("/:id/duplicate", g.Contributor(), g.KBAccessRead("id"), handler.DuplicateKnowledgeBase)
		// Bilgi bankası kopyalama ilerlemesini al — Viewer+; salt okunur. manage_kbs (copy işlemini başlatan anahtar) veya
		// retrieve ile yoklama yapılabilir; görevler kiracı bazında yalıtılır (requireTaskProgressTenant), anahtar yalnızca
		// kendi kiracısının görevlerini sorgulayabilir.
		kb.With(apiKeyRetrieve(apiKeyManageKnowledgeBases(apiKeyFullAccess()))).
			GET("/copy/progress/:task_id", g.Viewer(), handler.GetKBCloneProgress)
		// Taşınabilir hedef bilgi bankaları listesini al — Viewer+ ve KB için read izni gerekir
		kb.GET("/:id/move-targets", g.Viewer(), g.KBAccessRead("id"), handler.ListMoveTargets)
	}
}

// RegisterImageAttrRoutes wires the global, read-only image-attribute registry
// that drives the KB editor's attribute panel.
//
// The registry is a single source of truth, so adding an attribute is a
// backend-only change (one registry row) and the UI follows automatically. It
// carries no KB id and needs only the Viewer role, so — like the other
// read-only KB-editor helper GET /system/parser-engines — it is mounted at the top level rather than under
// the /knowledge-bases collection.
func RegisterImageAttrRoutes(r *gin.RouterGroup, handler *handler.KnowledgeBaseHandler, g *rbacGuards) {
	g.apiKeyRoute(r, http.MethodGet, "/image-attrs/schema",
		apiKeyRetrieve(apiKeyFullAccess()), g.Viewer(), handler.GetImageAttrsSchema)
}

// RegisterKnowledgeBaseActivityRoutes exposes the read-only per-KB activity
// feed. It intentionally stays JWT-only: audit history is a sensitive owner
// surface and no existing workspace API-key capability grants audit access.
func RegisterKnowledgeBaseActivityRoutes(r *gin.RouterGroup, auditHandler *handler.AuditLogHandler, g *rbacGuards) {
	if auditHandler == nil {
		return
	}
	r.GET("/knowledge-bases/:id/activity",
		g.OwnedKBOrAdmin(), g.KBAccessRead("id"), auditHandler.ListKnowledgeBaseActivity)
}

// RegisterKnowledgeTagRoutes, bilgi bankası etiketleriyle ilgili rotaları kaydeder.
//
// Tags are KB metadata: Viewer reads, Contributor writes. Per-KB
// ownership granularity for tags is out of scope for PR 2; this is
// purely role-based.
func RegisterKnowledgeTagRoutes(r *gin.RouterGroup, tagHandler *handler.TagHandler, g *rbacGuards) {
	if tagHandler == nil {
		return
	}
	// Tags, KB'nin alt kaynaklarıdır — etiket oluşturma/düzenleme/silme, KB içeriğinin arama sınıflandırmasını
	// değiştirir; KB ana gövdesiyle aynı "creator OR Admin+" matrisi uygulanmalıdır; böylece ilgisiz bir
	// Contributor'ın başkasının KB'sinde rastgele etiket oluşturup/silerek KB sahibinin içerik düzenini etkilemesi önlenir.
	kbTags := g.apiKeyGroup(r.Group("/knowledge-bases/:id/tags"), apiKeyIngest(apiKeyFullAccess()))
	kbTagsRead := kbTags.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// KBAccessRead/Write resolve own/shared/agent-visible access and
		// rewrite the request's tenant context to the effective tenant
		// for the duration of the handler — so the handler no longer
		// needs its own effectiveCtxForKB helper.
		kbTagsRead.GET("", g.Viewer(), g.KBAccessRead("id"), tagHandler.ListTags)
		kbTags.POST("", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.CreateTag)
		kbTags.PUT("/:tag_id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.UpdateTag)
		kbTags.DELETE("/:tag_id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.DeleteTag)
	}
}

// RegisterWikiPageRoutes registers wiki page related routes.
//
// Wiki pages are KB content (wiki mode): reads are Viewer+ and gated by
// KBAccessRead (own / org-shared / via shared agent), matching FAQ /
// chunk / tag read routes. Content mutations (create/update/delete) and
// maintenance actions (rebuild-links, auto-fix, change issue status)
// honour per-KB ownership via OwnedWikiKBOrAdmin (PR 5, #1303): the URL
// :kb_id resolves directly to the owning KB so a Contributor who owns
// the KB can manage its wiki, while a non-owner Contributor gets 403.
func RegisterWikiPageRoutes(r *gin.RouterGroup, wikiHandler *handler.WikiPageHandler, g *rbacGuards) {
	wiki := g.apiKeyGroup(r.Group("/knowledgebase/:kb_id/wiki"), apiKeyIngest(apiKeyFullAccess()))
	wikiRead := wiki.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Page CRUD
		wikiRead.GET("/pages", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.ListPages)
		wiki.POST("/pages", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.CreatePage)
		wiki.PUT("/move-page", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.MovePage)
		wikiRead.GET("/pages/*slug", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.GetPage)
		wiki.PUT("/pages/*slug", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.UpdatePage)
		wiki.DELETE("/pages/*slug", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.DeletePage)

		// Revision history (slug is a catch-all like /pages; revert carries
		// the slug in the body for the same reason move-page does)
		wikiRead.GET("/revisions/*slug", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.ListRevisions)
		wiki.POST("/revert", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.RevertPage)

		// Folder tree (directory nodes)
		wikiRead.GET("/folders", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.ListFolders)
		wiki.POST("/folders", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.CreateFolder)
		wiki.PUT("/folders/:folder_id", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.UpdateFolder)
		wiki.DELETE("/folders/:folder_id", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.DeleteFolder)

		// Special pages
		wikiRead.GET("/index", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.GetIndex)

		// Graph and stats
		wikiRead.GET("/graph", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.GetGraph)
		wikiRead.GET("/stats", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.GetStats)

		// Search and maintenance
		wikiRead.GET("/search", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.SearchPages)
		wiki.POST("/rebuild-links", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.RebuildLinks)
		wikiRead.GET("/lint", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.Lint)
		wiki.POST("/auto-fix", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.AutoFix)

		// Issues
		wikiRead.GET("/issues", g.Viewer(), g.KBAccessRead("kb_id"), wikiHandler.ListIssues)
		wiki.PUT("/issues/:issue_id/status", g.OwnedWikiKBOrAdmin(), g.KBAccessWrite("kb_id"), wikiHandler.UpdateIssueStatus)
	}

	wikiSearch := g.apiKeyGroup(r.Group("/wiki-search", g.Viewer()), apiKeyRetrieve(apiKeyFullAccess()))
	{
		wikiSearch.POST("", wikiHandler.SearchPagesAcross)
	}
}
