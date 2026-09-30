package session

import (
	stderrors "errors"
	"net/http"

	"github.com/acrbaran/rag/internal/application/service"
	"github.com/acrbaran/rag/internal/browserskill"
	"github.com/acrbaran/rag/internal/config"
	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/infrastructure/docparser"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Handler handles all HTTP requests related to conversation sessions
type Handler struct {
	browserSkill         *browserskill.Manager
	messageService       interfaces.MessageService // Service for managing messages
	suggestionService    interfaces.MessageSuggestionService
	sessionService       interfaces.SessionService       // Service for managing sessions
	streamManager        interfaces.StreamManager        // Manager for handling streaming responses
	config               *config.Config                  // Application configuration
	knowledgebaseService interfaces.KnowledgeBaseService // Service for managing knowledge bases
	customAgentService   interfaces.CustomAgentService   // Service for managing custom agents
	tenantService        interfaces.TenantService        // Service for loading tenant (shared agent context)
	agentShareService    interfaces.AgentShareService    // Service for resolving shared agents (KB scope in retrieval)
	kbShareService       interfaces.KBShareService       // Service for resolving shared KB permissions
	fileService          interfaces.FileService          // Service for file storage (image uploads)
	resourceCatalog      interfaces.ResourceCatalog
	storageResolver      interfaces.StorageBackendResolver
	modelService         interfaces.ModelService // Service for model management (VLM access)
	attachmentProcessor  *AttachmentProcessor    // Processor for file attachments
	temporaryDocuments   interfaces.TemporaryDocumentService
	// artifactCollector drains skill-generated files from the session sandbox
	// after an agent turn completes. May be nil when the sandbox backend does
	// not support artifact collection; handlers must check before using.
	artifactCollector *service.ArtifactCollector
	// workspaceCheckpointer commits the sandbox /workspace at the end of each
	// agent turn so session fork can roll back to a specific message. May be
	// nil when the deployment has no sandbox backend.
	workspaceCheckpointer *service.WorkspaceCheckpointer
	// sandboxIDLookup resolves a session's bound sandbox without provisioning.
	sandboxIDLookup SandboxIDLookup
	memoryService   interfaces.MemoryService // Service for cross-session long-term memory
	// userService / memberService back the sandbox terminal's self-contained
	// handshake (browser WebSocket upgrades cannot send Authorization).
	userService   interfaces.UserService
	memberService interfaces.TenantMemberService
	// terminalService opens PTYs on the sandbox bound to a session. It also
	// owns first-use provisioning: the WS handshake carries the chat page's
	// selected agent so the sandbox is created with the same config a
	// conversation turn would use.
	terminalService *service.SandboxTerminalService
	desktopService  *service.SandboxDesktopService
	desktopTickets  service.SandboxDesktopTicketStore
	desktopLast     service.SandboxDesktopLastStore
	// redis backs the distributed desktop slot. Nil in Lite mode, where the
	// in-process limiter is the correct degradation.
	redis *redis.Client
	// forkService branches a session at a chosen user message. May be nil in
	// deployments where fork is not wired; ForkSession checks.
	forkService sessionForker
	// rewindService truncates the current session at a chosen message. May
	// be nil in deployments where rewind is not wired; RewindSession checks.
	rewindService sessionRewinder
	// approvedProjectDirs is the user-approved ProjectDirs list used to
	// validate CreateSession's optional project_dir. Nil means none are
	// approved, so a non-empty project_dir is rejected.
	approvedProjectDirs HostProjectDirsLoader
}

// NewHandler creates a new instance of Handler with all necessary dependencies
func NewHandler(
	sessionService interfaces.SessionService,
	messageService interfaces.MessageService,
	suggestionService interfaces.MessageSuggestionService,
	streamManager interfaces.StreamManager,
	config *config.Config,
	knowledgebaseService interfaces.KnowledgeBaseService,
	customAgentService interfaces.CustomAgentService,
	tenantService interfaces.TenantService,
	agentShareService interfaces.AgentShareService,
	kbShareService interfaces.KBShareService,
	fileService interfaces.FileService,
	resourceCatalog interfaces.ResourceCatalog,
	storageResolver interfaces.StorageBackendResolver,
	modelService interfaces.ModelService,
	documentReader interfaces.DocumentReader,
	imageResolver *docparser.ImageResolver,
	temporaryDocuments interfaces.TemporaryDocumentService,
	artifactCollector *service.ArtifactCollector,
	workspaceCheckpointer *service.WorkspaceCheckpointer,
	sandboxIDLookup SandboxIDLookup,
	memoryService interfaces.MemoryService,
	userService interfaces.UserService,
	memberService interfaces.TenantMemberService,
	terminalService *service.SandboxTerminalService,
	browserSkill *browserskill.Manager,
	desktopService *service.SandboxDesktopService,
	desktopTickets service.SandboxDesktopTicketStore,
	desktopLast service.SandboxDesktopLastStore,
	rdb *redis.Client,
	forkService *service.SessionForkService,
	rewindService *service.SessionRewindService,
	approvedProjectDirs HostProjectDirsLoader,
) *Handler {
	h := &Handler{
		browserSkill:          browserSkill,
		sessionService:        sessionService,
		messageService:        messageService,
		suggestionService:     suggestionService,
		streamManager:         streamManager,
		config:                config,
		knowledgebaseService:  knowledgebaseService,
		customAgentService:    customAgentService,
		tenantService:         tenantService,
		agentShareService:     agentShareService,
		kbShareService:        kbShareService,
		fileService:           fileService,
		resourceCatalog:       resourceCatalog,
		storageResolver:       storageResolver,
		modelService:          modelService,
		temporaryDocuments:    temporaryDocuments,
		artifactCollector:     artifactCollector,
		workspaceCheckpointer: workspaceCheckpointer,
		sandboxIDLookup:       sandboxIDLookup,
		memoryService:         memoryService,
		userService:           userService,
		memberService:         memberService,
		terminalService:       terminalService,
		desktopService:        desktopService,
		desktopTickets:        desktopTickets,
		desktopLast:           desktopLast,
		redis:                 rdb,
		approvedProjectDirs:   approvedProjectDirs,
		attachmentProcessor: NewAttachmentProcessor(
			fileService,
			documentReader,
			imageResolver,
			modelService,
		),
	}
	if forkService != nil {
		h.forkService = forkService
	}
	if rewindService != nil {
		h.rewindService = rewindService
	}
	return h
}

// CreateSession godoc
// @Summary      Oturum oluştur
// @Description  Yeni bir konuşma oturumu oluştur
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        request  body      CreateSessionRequest  true  "Oturum oluşturma isteği"
// @Success      201      {object}  map[string]interface{}  "Oluşturulan oturum"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions [post]
func (h *Handler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	// Parse and validate the request body
	var request CreateSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error(ctx, "Failed to validate session creation parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Get tenant ID from context
	tenantID, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Failed to get tenant ID")
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	// Sessions are now knowledge-base-independent:
	// - All configuration comes from custom agent at query time
	// - Session only stores basic info (tenant ID, title, description)
	logger.Infof(
		ctx,
		"Processing session creation request, tenant ID: %d",
		tenantID,
	)

	hostDir, ok := bindHostWorkspaceDir(request.ProjectDir, h.approvedDirs())
	if !ok {
		_ = c.Error(errors.NewBadRequestError("project_dir is not an approved project directory"))
		return
	}

	// Create session object with base properties
	createdSession := &types.Session{
		TenantID:         tenantID.(uint64),
		Title:            request.Title,
		Description:      types.SanitizeClientSessionDescription(request.Description, ""),
		HostWorkspaceDir: hostDir,
	}
	// Attach the calling user as the session owner when available.
	// API-key callers scope sessions per external user when configured;
	// otherwise they fall back to the synthetic tenant user.
	if ownerID := types.SessionOwnerIDFromContext(ctx); ownerID != "" {
		createdSession.UserID = ownerID
	}

	// Call service to create session
	logger.Infof(ctx, "Calling session service to create session")
	createdSession, err := h.sessionService.CreateSession(ctx, createdSession)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return created session
	logger.Infof(ctx, "Session created successfully, ID: %s", createdSession.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdSession,
	})
}

// GetSession godoc
// @Summary      Oturum ayrıntılarını al
// @Description  ID'ye göre oturum ayrıntılarını al
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Oturum ID'si"
// @Success      200  {object}  map[string]interface{}  "Oturum ayrıntıları"
// @Failure      404  {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{id} [get]
func (h *Handler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start retrieving session")

	// Get session ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Call service to get session details
	logger.Infof(ctx, "Retrieving session, ID: %s", id)
	session, err := h.sessionService.GetSession(ctx, id)
	if err != nil {
		if stderrors.Is(err, errors.ErrSessionNotFound) {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return session data
	logger.Infof(ctx, "Session retrieved successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    session,
	})
}

// GetSessionsByTenant godoc
// @Summary      Oturum listesini al
// @Description  Geçerli alanın oturum listesini al; sayfalama, anahtar kelime araması ve kaynak/Agent filtrelemeyi destekler
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        page       query     int     false  "Sayfa numarası"
// @Param        page_size  query     int     false  "Sayfa başına öğe sayısı"
// @Param        keyword    query     string  false  "Başlıkta bulanık arama"
// @Param        source     query     string  false  "Kaynak filtresi: web / embed / api / wechat / slack / telegram / ...（api、embed、IM kanalları Admin+ gerektirir）"
// @Param        agent_id   query     string  false  "Agent'e göre filtrele（yalnızca IM oturumları için geçerlidir）"
// @Success      200        {object}  map[string]interface{}  "Oturum listesi"
// @Failure      400        {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions [get]
func (h *Handler) GetSessionsByTenant(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse pagination parameters from query
	var pagination types.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		logger.Error(ctx, "Failed to parse pagination parameters", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Response items always include pin state and (when available) IM origin
	// fields so the frontend can render pin icons / source badges without a
	// second roundtrip. Unset filter params behave like "no filter".
	result, err := h.sessionService.ListSessions(ctx, &types.SessionListQuery{
		Keyword:  c.Query("keyword"),
		Source:   c.Query("source"),
		AgentID:  c.Query("agent_id"),
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	})
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result.Data,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

// UpdateSession godoc
// @Summary      Oturumu güncelle
// @Description  Oturum özelliklerini güncelle
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        id       path      string         true  "Oturum ID'si"
// @Param        request  body      types.Session  true  "Oturum bilgileri"
// @Success      200      {object}  map[string]interface{}  "Güncellenmiş oturum"
// @Failure      404      {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{id} [put]
func (h *Handler) UpdateSession(c *gin.Context) {
	ctx := c.Request.Context()

	// Get session ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Verify tenant ID from context for authorization
	tenantID, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Failed to get tenant ID")
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	// Parse request body to session object
	var session types.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		logger.Error(ctx, "Failed to parse session data", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	session.ID = id
	session.TenantID = tenantID.(uint64)

	// Call service to update session
	if err := h.sessionService.UpdateSession(ctx, &session); err != nil {
		if stderrors.Is(err, errors.ErrSessionNotFound) {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Reload session from database to return complete timestamps and stored fields
	updatedSession, err := h.sessionService.GetSession(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Return updated session
	logger.Infof(ctx, "Session updated successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedSession,
	})
}

// DeleteSession godoc
// @Summary      Oturumu sil
// @Description  Belirtilen oturumu sil
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Oturum kimliği"
// @Success      200  {object}  map[string]interface{}  "Başarıyla silindi"
// @Failure      404  {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{id} [delete]
func (h *Handler) DeleteSession(c *gin.Context) {
	ctx := c.Request.Context()

	// Get session ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	// Call service to delete session
	if err := h.sessionService.DeleteSession(ctx, id); err != nil {
		if stderrors.Is(err, errors.ErrSessionNotFound) {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	h.browserSkill.Forget(browserSkillScope(ctx), []string{id})

	// Return success message
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session deleted successfully",
	})
}

// ClearSessionMessages godoc
// @Summary      Oturum mesajlarını temizle
// @Description  Oturumdaki tüm mesajları siler; aynı zamanda LLM bağlamını ve sohbet geçmişi bilgi bankası girdilerini temizler. Oturumun kendisi korunur.
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Oturum kimliği"
// @Success      200  {object}  map[string]interface{}  "Başarıyla temizlendi"
// @Failure      400  {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Failure      404  {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{id}/messages [delete]
func (h *Handler) ClearSessionMessages(c *gin.Context) {
	ctx := c.Request.Context()

	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	logger.Infof(ctx, "Clearing all messages for session: %s", id)

	if err := h.messageService.ClearSessionMessages(ctx, id); err != nil {
		if stderrors.Is(err, errors.ErrSessionNotFound) {
			logger.Warnf(ctx, "Session not found, ID: %s", id)
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"session_id": id})
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Session messages cleared successfully, ID: %s", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session messages cleared successfully",
	})
}

// batchDeleteRequest represents the request body for batch deleting sessions
type batchDeleteRequest struct {
	IDs       []string `json:"ids"`
	DeleteAll bool     `json:"delete_all"`
}

// BatchDeleteSessions godoc
// @Summary      Oturumları toplu sil
// @Description  Kimlik listesine göre sohbet oturumlarını toplu olarak siler veya geçerli alandaki tüm oturumları silmek için delete_all=true ayarlanır
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        request  body      batchDeleteRequest  true  "Toplu silme isteği"
// @Success      200      {object}  map[string]interface{}  "Silme sonucu"
// @Failure      400      {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/batch [delete]
func (h *Handler) BatchDeleteSessions(c *gin.Context) {
	ctx := c.Request.Context()

	var req batchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(ctx, "Invalid batch delete request: %v", err)
		c.Error(errors.NewBadRequestError("invalid request"))
		return
	}

	if req.DeleteAll {
		if err := h.sessionService.DeleteAllSessions(ctx); err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(errors.NewInternalServerError(err.Error()))
			return
		}
		h.browserSkill.ForgetAll(browserSkillScope(ctx))
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "All sessions deleted successfully",
		})
		return
	}

	if len(req.IDs) == 0 {
		c.Error(errors.NewBadRequestError("ids are required when delete_all is false"))
		return
	}

	// Sanitize all IDs
	sanitizedIDs := make([]string, 0, len(req.IDs))
	for _, id := range req.IDs {
		sanitized := secutils.SanitizeForLog(id)
		if sanitized != "" {
			sanitizedIDs = append(sanitizedIDs, sanitized)
		}
	}

	if len(sanitizedIDs) == 0 {
		c.Error(errors.NewBadRequestError("no valid session IDs provided"))
		return
	}

	if err := h.sessionService.BatchDeleteSessions(ctx, sanitizedIDs); err != nil {
		if stderrors.Is(err, errors.ErrSessionNotFound) {
			logger.Warnf(ctx, "No visible sessions found for batch delete")
			c.Error(errors.NewNotFoundError(err.Error()))
			return
		}
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	h.browserSkill.Forget(browserSkillScope(ctx), sanitizedIDs)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sessions deleted successfully",
	})
}

// PinSession godoc
// @Summary      Oturumu sabitle
// @Description  Belirtilen oturumu sabitler (kullanıcı düzeyinde)
// @Tags         Oturumlar
// @Produce      json
// @Param        session_id   path      string  true  "Oturum kimliği"
// @Success      200  {object}  map[string]interface{}  "Başarıyla sabitlendi"
// @Failure      404  {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{session_id}/pin [post]
func (h *Handler) PinSession(c *gin.Context) {
	h.setSessionPinned(c, true)
}

// UnpinSession godoc
// @Summary      Oturum sabitlemesini kaldır
// @Description  Belirtilen oturumun sabitlemesini kaldır
// @Tags         Oturumlar
// @Produce      json
// @Param        id   path      string  true  "Oturum ID'si"
// @Success      200  {object}  map[string]interface{}  "Sabitleme iptal edildi"
// @Failure      404  {object}  errors.AppError         "Oturum bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{id}/pin [delete]
func (h *Handler) UnpinSession(c *gin.Context) {
	h.setSessionPinned(c, false)
}

func (h *Handler) setSessionPinned(c *gin.Context, pinned bool) {
	ctx := c.Request.Context()

	// POST and DELETE for /sessions/.../pin register under different wildcards
	// (POST :session_id, DELETE :id — see router.go). Accept whichever is set.
	rawID := c.Param("session_id")
	if rawID == "" {
		rawID = c.Param("id")
	}
	id := secutils.SanitizeForLog(rawID)
	if id == "" {
		logger.Error(ctx, "Session ID is empty")
		c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}

	rows, err := h.sessionService.SetSessionPinned(ctx, id, pinned)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": id,
			"pinned":     pinned,
		})
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}
	// Zero rows means the session doesn't exist or isn't visible to this user;
	// tell the client rather than reporting success.
	if rows == 0 {
		c.Error(errors.NewNotFoundError(errors.ErrSessionNotFound.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"is_pinned": pinned,
	})
}
