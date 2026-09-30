package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/application/service"
	"github.com/acrbaran/rag/internal/config"
	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/handler/dto"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
)

const (
	oidcNonceCookieName   = "rethra_oidc_nonce"
	oidcNonceCookieMaxAge = 600
)

// AuthHandler implements HTTP request handlers for user authentication
// Provides functionality for user registration, login, logout, and token management
// through the REST API endpoints
type AuthHandler struct {
	userService      interfaces.UserService
	tenantService    interfaces.TenantService
	configInfo       *config.Config
	systemSettingSvc interfaces.SystemSettingService
	// invitationSvc is required for the share-link registration path
	// (POST /auth/register-by-invite). When nil — e.g. legacy test
	// fixtures — the share-link endpoints respond 503 rather than
	// blocking the rest of the auth surface.
	invitationSvc interfaces.TenantInvitationService
}

// NewAuthHandler creates a new auth handler instance with the provided services
// Parameters:
//   - userService: An implementation of the UserService interface for business logic
//   - tenantService: An implementation of the TenantService interface for tenant management
//   - systemSettingSvc: 3-tier resolver for runtime-tunable settings such as
//     auth.registration_mode (P3). When DB has a row, it overrides cfg's
//     startup value; otherwise we fall back to cfg.Auth.RegistrationMode
//     (which already accounted for the legacy DISABLE_REGISTRATION env coerce
//     during config load). Mismatch impossible by construction since the
//     handler always passes cfg's value as the def parameter to GetString.
//
// Returns a pointer to the newly created AuthHandler
func NewAuthHandler(configInfo *config.Config,
	userService interfaces.UserService, tenantService interfaces.TenantService,
	systemSettingSvc interfaces.SystemSettingService,
	invitationSvc interfaces.TenantInvitationService,
) *AuthHandler {
	// Boot-time guard: a nil-or-empty Auth section silently disables the
	// invite_only gate (see Register below). Emit a loud one-shot log
	// pointing at the misconfiguration so operators notice on startup
	// instead of discovering it the day someone hits /auth/register.
	if configInfo == nil || configInfo.Auth == nil {
		logger.Errorf(context.Background(),
			"[auth] AuthHandler constructed with nil/incomplete config (cfg=%v); "+
				"registration_mode enforcement is disabled. This is almost certainly a wiring bug.",
			configInfo)
	}
	return &AuthHandler{
		configInfo:       configInfo,
		userService:      userService,
		tenantService:    tenantService,
		systemSettingSvc: systemSettingSvc,
		invitationSvc:    invitationSvc,
	}
}

// resolveRegistrationMode returns the currently active registration mode.
// Priority: DB system_settings > cfg (which already absorbed the legacy
// DISABLE_REGISTRATION env coerce at startup) > "self_serve" hard default.
//
// Centralised here so /auth/register and /auth/config stay in lock-step —
// otherwise a SystemAdmin's UI edit could affect one path and not the other.
func (h *AuthHandler) resolveRegistrationMode(ctx context.Context) string {
	// cfg-derived default: empty is impossible after applyAuthAndTenantDefaults,
	// but be defensive in case AuthHandler was constructed before that ran
	// (the NewAuthHandler guard already logged in that case).
	def := config.AuthRegistrationModeSelfServe
	if h.configInfo != nil && h.configInfo.Auth != nil {
		if m := strings.TrimSpace(h.configInfo.Auth.RegistrationMode); m != "" {
			def = m
		}
	}
	if h.systemSettingSvc == nil {
		return def
	}
	// envName = "" because DISABLE_REGISTRATION is a boolean and
	// auth.registration_mode is a string — the legacy env was already
	// coerced into `def` above. Mixing the two semantics at the resolver
	// layer would mean a UI delete (DB row absent) silently flipped to
	// the legacy boolean read again, which is surprising.
	return h.systemSettingSvc.GetString(ctx, "auth.registration_mode", "", def)
}

// resolveDefaultTenantMode returns the provisioning policy for a new
// local user account.
// Priority: DB system_settings > cfg.Auth > hard default (create_personal).
// Shared by public registration and the SystemAdmin create-user endpoint.
//
// Invitation registration never uses this value: the invitation itself
// supplies the target tenant.
func resolveDefaultTenantMode(
	ctx context.Context,
	configInfo *config.Config,
	systemSettingSvc interfaces.SystemSettingService,
) types.TenantProvisioningMode {
	def := config.AuthDefaultTenantModeCreatePersonal
	if configInfo != nil && configInfo.Auth != nil {
		if mode := strings.TrimSpace(configInfo.Auth.DefaultTenantMode); mode != "" {
			def = mode
		}
	}
	mode := def
	if systemSettingSvc != nil {
		mode = systemSettingSvc.GetString(
			ctx,
			"auth.default_tenant_mode",
			"RETHRA_AUTH_DEFAULT_TENANT_MODE",
			def,
		)
	}
	if mode == config.AuthDefaultTenantModeTenantless {
		return types.TenantProvisioningTenantless
	}
	return types.TenantProvisioningCreatePersonal
}

// resolveDefaultTenantMode returns the provisioning policy for ordinary
// public password registrations.
func (h *AuthHandler) resolveDefaultTenantMode(ctx context.Context) types.TenantProvisioningMode {
	return resolveDefaultTenantMode(ctx, h.configInfo, h.systemSettingSvc)
}

// resolveDefaultTenantMode resolves the same policy for users provisioned
// by a SystemAdmin via POST /api/v1/system/admin/users/create.
func (h *SystemHandler) resolveDefaultTenantMode(ctx context.Context) types.TenantProvisioningMode {
	return resolveDefaultTenantMode(ctx, h.cfg, h.systemSettingSvc)
}

func (h *AuthHandler) complexPasswordEnabled(ctx context.Context) bool {
	return service.ResolveComplexPasswordEnabled(ctx, h.configInfo, h.systemSettingSvc)
}

func (h *SystemHandler) complexPasswordEnabled(ctx context.Context) bool {
	return service.ResolveComplexPasswordEnabled(ctx, h.cfg, h.systemSettingSvc)
}

// Register godoc
// @Summary      Kullanıcı kaydı
// @Description  Yeni kullanıcı hesabı kaydet
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      types.RegisterRequest  true  "Kayıt isteği parametreleri"
// @Success      201      {object}  types.RegisterResponse
// @Failure      400      {object}  errors.AppError  "İstek parametresi hatası"
// @Failure      403      {object}  errors.AppError  "Kayıt özelliği devre dışı"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user registration")

	// auth.registration_mode=invite_only olduğunda public kayıt kapatılır.
	// Öncelik: DB system_settings > cfg.Auth.RegistrationMode > "self_serve".
	// SystemAdmin, self_serve / invite_only seçeneğini “Genel Ayarlar” UI üzerinden gerçek zamanlı olarak değiştirir, hemen
	// etkili olur; hizmetin yeniden başlatılması gerekmez. Eski DISABLE_REGISTRATION=true değişkeni config
	// başlangıç aşamasında eşdeğer biçimde invite_only değerine yükseltilmeye devam eder (applyAuthAndTenantDefaults),
	// ve cfg-default olarak resolveRegistrationMode içine girer.
	if h.resolveRegistrationMode(ctx) == config.AuthRegistrationModeInviteOnly {
		logger.Warn(ctx, "Registration rejected: auth.registration_mode=invite_only")
		appErr := errors.NewForbiddenError("Registration is invite-only")
		c.Error(appErr)
		return
	}

	var req types.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse registration request parameters", err)
		appErr := errors.NewValidationError("Invalid registration parameters").WithDetails(err.Error())
		c.Error(appErr)
		return
	}
	req.Username = secutils.SanitizeForLog(req.Username)
	req.Email = secutils.SanitizeForLog(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.FirstName == "" || req.LastName == "" || req.Phone == "" {
		c.Error(errors.NewValidationError("first name, last name and phone are required"))
		return
	}
	// Password is intentionally NOT sanitized: SanitizeForLog replaces
	// \n, \r, \t and strips other control characters so a string is safe
	// to write into a log line. Applying it to a real password would
	// silently rewrite the credential before hashing, so registration
	// would succeed but login with the original password would fail.
	// Passwords must never be logged, so they don't need that defence.

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		logger.Error(ctx, "Missing required registration fields")
		appErr := errors.NewValidationError("Username, email and password are required")
		c.Error(appErr)
		return
	}

	// Validate password against the runtime policy (DB system_settings
	// first). Register itself does not enforce this because OIDC
	// auto-provision uses an untyped random secret the user never types.
	if err := service.ValidatePasswordPolicy(req.Password, h.complexPasswordEnabled(ctx)); err != nil {
		logger.Error(ctx, "Invalid password policy")
		appErr := errors.NewValidationError(err.Error())
		_ = c.Error(appErr)
		return
	}

	req.Username = secutils.SanitizeForLog(req.Username)
	req.Email = secutils.SanitizeForLog(req.Email)
	req.TenantProvisioning = h.resolveDefaultTenantMode(ctx)
	// Call service to register user
	user, err := h.userService.Register(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to register user: %v", err)
		appErr := errors.NewBadRequestError(err.Error())
		c.Error(appErr)
		return
	}

	// Return success response
	response := &types.RegisterResponse{
		Success: true,
		Message: "Registration successful",
		User:    user,
	}

	logger.Infof(ctx, "User registered successfully: %s", secutils.SanitizeForLog(user.Email))
	c.JSON(http.StatusCreated, response)
}

// Login godoc
// @Summary      Kullanıcı girişi
// @Description  Kullanıcı giriş yapar ve erişim belirteci alır
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      types.LoginRequest  true  "Giriş isteği parametreleri"
// @Success      200      {object}  types.LoginResponse
// @Failure      401      {object}  errors.AppError  "Kimlik doğrulama başarısız"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user login")

	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse login request parameters", err)
		appErr := errors.NewValidationError("Invalid login parameters").WithDetails(err.Error())
		c.Error(appErr)
		return
	}
	email := secutils.SanitizeForLog(req.Email)

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		logger.Error(ctx, "Missing required login fields")
		appErr := errors.NewValidationError("Email and password are required")
		c.Error(appErr)
		return
	}

	// Call service to authenticate user
	response, err := h.userService.Login(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to login user: %v", err)
		appErr := errors.NewUnauthorizedError("Login failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Check if login was successful
	if !response.Success {
		logger.Warnf(ctx, "Login failed: %s", response.Message)
		c.JSON(http.StatusUnauthorized, dto.NewAuthLoginResponse(response))
		return
	}

	// User is already in the correct format from service

	logger.Infof(ctx, "User logged in successfully, email: %s", email)
	c.JSON(http.StatusOK, dto.NewAuthLoginResponse(response))
}

// GetOIDCAuthorizationURL godoc
// @Summary      OIDC yetkilendirme adresini al
// @Description  Arka uç OIDC yapılandırmasına göre üçüncü taraf giriş yönlendirme adresi oluştur
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        redirect_uri  query     string  true  "OIDC geri çağırma adresi"
// @Success      200           {object}  types.OIDCAuthURLResponse
// @Failure      400           {object}  errors.AppError  "İstek parametresi hatası"
// @Failure      403           {object}  errors.AppError  "OIDC etkin değil"
// @Router       /auth/oidc/url [get]
func (h *AuthHandler) GetOIDCAuthorizationURL(c *gin.Context) {
	ctx := c.Request.Context()
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	if redirectURI == "" {
		appErr := errors.NewValidationError("redirect_uri is required")
		c.Error(appErr)
		return
	}

	resp, err := h.userService.GetOIDCAuthorizationURL(ctx, redirectURI)
	if err != nil {
		logger.Errorf(ctx, "Failed to generate OIDC authorization URL: %v", err)
		appErr := errors.NewForbiddenError("OIDC authorization unavailable").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Bind the state nonce to this browser so an attacker cannot replay
	// their own authorization code into a victim's callback.
	setOIDCNonceCookie(c, resp.Nonce)

	c.JSON(http.StatusOK, resp)
}

// setOIDCNonceCookie binds the OIDC state nonce to this browser so an
// attacker cannot replay their own authorization code into a victim's
// callback. Shared by /auth/oidc/url (JSON) and /auth/oidc/start (302).
func setOIDCNonceCookie(c *gin.Context, nonce string) {
	if nonce == "" {
		return
	}
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oidcNonceCookieName, nonce, oidcNonceCookieMaxAge, "/", "", secure, true)
}

// oidcCallbackURL derives the absolute /auth/oidc/callback URL from the
// request's own origin (scheme + host), so external platforms can deep-link
// to /auth/oidc/start without supplying a redirect_uri.
func oidcCallbackURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/api/v1/auth/oidc/callback"
}

// OIDCStart godoc
// @Summary      OIDC girişini başlat (doğrudan 302)
// @Description  /auth/oidc/url adresinden farklı olarak bu uç nokta doğrudan OIDC Provider yetkilendirme sayfasına 302 ile yönlendirir,
// @Description  Ön uç JS müdahalesi gerekmez. Harici platformların (ör. kurumsal portal) doğrudan bir bağlantı sunması için uygundur
// @Description  IdP'nin SSO oturumu sayesinde parolayı yeniden girmeden OIDC yetkilendirme kodu akışını tetikler.
// @Tags         Kimlik Doğrulama
// @Success      302
// @Router       /auth/oidc/start [get]
func (h *AuthHandler) OIDCStart(c *gin.Context) {
	ctx := c.Request.Context()
	resp, err := h.userService.GetOIDCAuthorizationURL(ctx, oidcCallbackURL(c))
	if err != nil {
		logger.Errorf(ctx, "Failed to generate OIDC authorization URL: %v", err)
		appErr := errors.NewForbiddenError("OIDC authorization unavailable").WithDetails(err.Error())
		c.Error(appErr)
		return
	}
	setOIDCNonceCookie(c, resp.Nonce)
	c.Redirect(http.StatusFound, resp.AuthorizationURL)
}

// GetOIDCConfig godoc
// @Summary      OIDC giriş yapılandırmasını al
// @Description  Ön yüzün OIDC giriş seçeneğini gösterip göstermeyeceğine karar vermesi için OIDC'nin etkin olup olmadığını ve provider görüntüleme adını döndürür
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Success      200  {object}  types.OIDCConfigResponse
// @Router       /auth/oidc/config [get]
func (h *AuthHandler) GetOIDCConfig(c *gin.Context) {
	providerDisplayName := ""
	enabled := false

	if h.configInfo != nil && h.configInfo.OIDCAuth != nil {
		enabled = h.configInfo.OIDCAuth.Enable
		providerDisplayName = strings.TrimSpace(h.configInfo.OIDCAuth.ProviderDisplayName)
	}

	c.JSON(http.StatusOK, &types.OIDCConfigResponse{
		Success:             true,
		Enabled:             enabled,
		ProviderDisplayName: providerDisplayName,
	})
}

// OIDCRedirectCallback godoc
// @Summary      OIDC giriş yönlendirme geri çağrısı
// @Description  OIDC provider geri çağrısını alır, kod değişimini arka uçta tamamlar ve ardından ön uç giriş sayfasına yönlendirir
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        code   query string false "OIDC yetkilendirme kodu"
// @Param        state  query string false "OIDC durumu"
// @Param        error  query string false "OIDC hata kodu"
// @Success      302
// @Router       /auth/oidc/callback [get]
func (h *AuthHandler) OIDCRedirectCallback(c *gin.Context) {
	ctx := c.Request.Context()
	frontendRedirectURI := "/"

	if providerError := strings.TrimSpace(c.Query("error")); providerError != "" {
		redirectURL := frontendRedirectURI + "#oidc_error=" + urlQueryEscape(providerError)
		if description := strings.TrimSpace(c.Query("error_description")); description != "" {
			redirectURL += "&oidc_error_description=" + urlQueryEscape(description)
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	decodedState, err := decodeOIDCState(state, c.Request)
	if err != nil {
		logger.Errorf(ctx, "Failed to decode OIDC state: %v", err)
		c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_error="+urlQueryEscape("invalid_state"))
		return
	}
	// One-time use: clear the binding cookie as soon as it is checked.
	c.SetCookie(oidcNonceCookieName, "", -1, "/", "", false, true)

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_error="+urlQueryEscape("missing_code"))
		return
	}

	resp, err := h.userService.LoginWithOIDC(ctx, code, strings.TrimSpace(decodedState.RedirectURI), h.resolveDefaultTenantMode(ctx))
	if err != nil {
		logger.Errorf(ctx, "Failed to complete OIDC login via redirect callback: %v", err)
		c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_error="+urlQueryEscape("login_failed")+"&oidc_error_description="+urlQueryEscape(err.Error()))
		return
	}
	if !resp.Success {
		c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_error="+urlQueryEscape("login_failed")+"&oidc_error_description="+urlQueryEscape(resp.Message))
		return
	}

	payload, err := encodeOIDCCallbackPayload(resp)
	if err != nil {
		logger.Errorf(ctx, "Failed to encode OIDC callback payload: %v", err)
		c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_error="+urlQueryEscape("payload_encode_failed"))
		return
	}

	c.Redirect(http.StatusFound, frontendRedirectURI+"#oidc_result="+urlQueryEscape(payload))
}

func encodeOIDCCallbackPayload(resp *types.OIDCCallbackResponse) (string, error) {
	payload, err := json.Marshal(dto.NewAuthOIDCCallbackResponse(resp))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

type oidcStatePayload struct {
	Nonce       string
	RedirectURI string
}

func decodeOIDCState(raw string, req *http.Request) (*oidcStatePayload, error) {
	payload, err := secutils.VerifyOIDCState(raw)
	if err != nil {
		return nil, err
	}
	cookieNonce, err := req.Cookie(oidcNonceCookieName)
	if err != nil || cookieNonce == nil || strings.TrimSpace(cookieNonce.Value) == "" {
		return nil, errors.NewValidationError("oidc nonce cookie missing")
	}
	if cookieNonce.Value != payload.Nonce {
		return nil, errors.NewValidationError("oidc nonce mismatch")
	}
	return &oidcStatePayload{
		Nonce:       payload.Nonce,
		RedirectURI: strings.TrimSpace(payload.RedirectURI),
	}, nil
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"#", "%23",
		"&", "%26",
		"+", "%2B",
		"=", "%3D",
		"?", "%3F",
	)
	return replacer.Replace(value)
}

// Logout godoc
// @Summary      Kullanıcı çıkışı
// @Description  Geçerli erişim belirtecini iptal eder ve çıkış yapar
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Çıkış başarılı"
// @Failure      400  {object}  errors.AppError         "İstek parametreleri hatalı"
// @Security     Bearer
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start user logout")

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger.Error(ctx, "Missing Authorization header")
		appErr := errors.NewValidationError("Authorization header is required")
		c.Error(appErr)
		return
	}

	// Parse Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Error(ctx, "Invalid Authorization header format")
		appErr := errors.NewValidationError("Invalid Authorization header format")
		c.Error(appErr)
		return
	}

	token := tokenParts[1]

	// Revoke every outstanding session for this user so refresh tokens
	// cannot keep working after logout.
	err := h.userService.Logout(ctx, token)
	if err != nil {
		logger.Errorf(ctx, "Failed to revoke token: %v", err)
		appErr := errors.NewInternalServerError("Logout failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Info(ctx, "User logged out successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

// RefreshToken godoc
// @Summary      Belirteci yenile
// @Description  Yeni bir erişim belirteci almak için yenileme belirtecini kullanır
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      object{refreshToken=string}  true  "Yenileme belirteci"
// @Success      200      {object}  map[string]interface{}       "Yeni belirteç"
// @Failure      401      {object}  errors.AppError              "Belirteç geçersiz"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start token refresh")

	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse refresh token request", err)
		appErr := errors.NewValidationError("Invalid refresh token request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Call service to refresh token
	accessToken, newRefreshToken, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logger.Errorf(ctx, "Failed to refresh token: %v", err)
		appErr := errors.NewUnauthorizedError("Token refresh failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Info(ctx, "Token refreshed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Token refreshed successfully",
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// GetCurrentUser godoc
// @Summary      Geçerli kullanıcı bilgilerini al
// @Description  Geçerli oturum açmış kullanıcının ayrıntılı bilgilerini alır
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Kullanıcı bilgileri"
// @Failure      401  {object}  errors.AppError         "Yetkisiz"
// @Security     Bearer
// @Router       /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Get current user from service (which extracts from context)
	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to get current user: %v", err)
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Get tenant information for the *active* tenant (the one the
	// auth middleware resolved against the X-Tenant-ID header), not
	// the user's home tenant. user.TenantID is the row stored on the
	// users table at signup time and never changes; reading it here
	// would make /auth/me always return the home tenant even after
	// the user switched into a peer tenant. The frontend then re-keys
	// `authStore.tenant.id` to the home tenant, and every UI gate
	// computed against it (currentTenantRole, isOwner, ...) leaks
	// the wrong role. Pull the active tenant id from context instead.
	var tenant *types.Tenant
	activeTenantID, _ := types.TenantIDFromContext(ctx)
	if activeTenantID == 0 {
		activeTenantID = user.TenantID
	}
	if activeTenantID > 0 {
		tenant, err = h.tenantService.GetTenantByID(ctx, activeTenantID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get tenant info for user %s, tenant ID %d: %v", user.Email, activeTenantID, err)
			// Don't fail the request if tenant info is not available
		}
	}
	userInfo := user.ToUserInfo()
	userInfo.CanAccessAllTenants = user.CanAccessAllTenants && h.configInfo.Tenant.EnableCrossTenantAccess
	// Mevcut kullanıcının memberships bilgisini eşzamanlı döndürür; böylece ön yüz sayfa yenilendiğinde (yalnızca /auth/me çağrıldığında)
	// sonrasında da currentTenantRole geri yüklenebilir ve rol bilgilerinin yalnızca login anında kullanılabilir olması önlenir.
	memberships := h.userService.BuildLoginMemberships(ctx, user, tenant)
	canCreateTenant := user.CanAccessAllTenants ||
		resolveTenantSelfServiceCreationEnabled(ctx, h.configInfo, h.systemSettingSvc)
	autoAcceptInvitation := h.systemSettingSvc != nil &&
		h.systemSettingSvc.GetBool(ctx, "tenant.auto_accept_invitation", "RETHRA_TENANT_AUTO_ACCEPT_INVITATION", false)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user":                userInfo,
			"preference_defaults": gin.H{"browser_search_instructions": types.DefaultBrowserSearchInstructions},
			"tenant":              dto.NewTenantResponse(ctx, tenant),
			"memberships":         memberships,
			"tenant_required":     tenant == nil,
			"capabilities": gin.H{
				"can_create_tenant":      canCreateTenant,
				"auto_accept_invitation": autoAcceptInvitation,
			},
		},
	})
}

// updateMyPreferencesRequest is the body for PUT /auth/me/preferences.
// Fields are pointers so the handler can distinguish "key not present"
// (preserve existing value) from "explicit false". See
// types.UserPreferences for the persistence-layer counterpart.
type updateMyPreferencesRequest struct {
	BrowserSearchInstructions *string `json:"browser_search_instructions" binding:"omitempty,max=4000"`
	// LastActiveTenantID lets clients persist "after a fresh login,
	// drop me back into this workspace" across devices. The SPA sends
	// this after every tenant switch; POST /auth/switch-tenant records
	// the same preference server-side. Send a positive workspace id to
	// set / replace, or 0 to clear. Membership is validated at next
	// login, not here. Nil = field omitted from the PATCH and stays
	// untouched.
	LastActiveTenantID *uint64 `json:"last_active_tenant_id"`
}

// UpdateMyPreferences godoc
// @Summary      Geçerli kullanıcının kişiselleştirme ayarlarını güncelle
// @Description  Kullanıcı tercihlerini PATCH anlamına göre birleştirir (yalnızca istek gövdesinde bulunan alanların üzerine yazılır, diğer alanlar değişmeden kalır),
// @Description  Veriler users.preferences (JSON) içinde saklanır ve cihazlar/tarayıcılar arasında otomatik olarak senkronize edilir.
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      updateMyPreferencesRequest  true  "Preferences patch"
// @Success      200      {object}  map[string]interface{}      "Güncellenmiş tercihler"
// @Failure      400      {object}  errors.AppError             "Hatalı istek parametreleri"
// @Failure      401      {object}  errors.AppError             "Yetkisiz"
// @Security     Bearer
// @Router       /auth/me/preferences [put]
func (h *AuthHandler) UpdateMyPreferences(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	var req updateMyPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid preferences request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	patch := types.UserPreferences{
		LastActiveTenantID:        req.LastActiveTenantID,
		BrowserSearchInstructions: req.BrowserSearchInstructions,
	}
	prefs, err := h.userService.UpdateUserPreferences(ctx, user.ID, patch)
	if err != nil {
		logger.Errorf(ctx, "Failed to update preferences for user %s: %v", user.Email, err)
		appErr := errors.NewBadRequestError("Failed to update preferences").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    prefs,
	})
}

// ChangePassword godoc
// updateMyProfileRequest is the body for PUT /auth/me/profile.
type updateMyProfileRequest struct {
	FirstName string `json:"first_name" binding:"required,max=100"`
	LastName  string `json:"last_name" binding:"required,max=100"`
	Phone     string `json:"phone" binding:"required,max=16"`
}

// UpdateMyProfile godoc
// @Summary      Update the current user's profile
// @Description  Updates first name, last name and phone (E.164) of the calling user.
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      updateMyProfileRequest  true  "Profile update"
// @Success      200      {object}  map[string]interface{}  "Updated user"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Failure      401      {object}  errors.AppError         "Yetkisiz"
// @Security     Bearer
// @Router       /auth/me/profile [put]
func (h *AuthHandler) UpdateMyProfile(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error()))
		return
	}

	var req updateMyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid profile request").WithDetails(err.Error()))
		return
	}

	updated, err := h.userService.UpdateMyProfile(ctx, user.ID, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		logger.Errorf(ctx, "Failed to update profile for user %s: %v", user.ID, err)
		c.Error(errors.NewBadRequestError("Failed to update profile").WithDetails(err.Error()))
		return
	}

	userInfo := updated.ToUserInfo()
	userInfo.CanAccessAllTenants = updated.CanAccessAllTenants && h.configInfo.Tenant.EnableCrossTenantAccess
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"user": userInfo},
	})
}

// ChangePassword godoc
// @Summary      Parolayı değiştir
// @Description  Geçerli kullanıcının giriş parolasını değiştirir. Yeni parola 8–32 karakter uzunluğunda olmalı ve hem harf hem sayı içermelidir; karmaşık parola etkinleştirildiğinde büyük/küçük harf ve özel karakter de içermelidir. Başarılı olduktan sonra tüm oturumlar iptal edilir ve yeniden giriş yapılması gerekir.
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      object{old_password=string,new_password=string}  true  "Parola değiştirme isteği"
// @Success      200      {object}  map[string]interface{}                           "Başarıyla değiştirildi"
// @Failure      400      {object}  errors.AppError                                  "Hatalı istek parametreleri"
// @Security     Bearer
// @Router       /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start password change")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse password change request", err)
		appErr := errors.NewValidationError("Invalid password change request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Get current user
	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to get current user: %v", err)
		appErr := errors.NewUnauthorizedError("Failed to get user information").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	// Change password. Policy is enforced in the service after the old
	// password is verified so a wrong current credential is not masked
	// by a complexity error.
	err = h.userService.ChangePassword(ctx, user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case service.IsPasswordPolicyError(err):
			appErr := errors.NewValidationError("Password policy violation").
				WithDetails(service.DetailPasswordPolicy)
			_ = c.Error(appErr)
			return
		case stderrors.Is(err, service.ErrInvalidOldPassword):
			appErr := errors.NewBadRequestError("Current password is incorrect").
				WithDetails(service.DetailInvalidOldPassword)
			c.Error(appErr)
			return
		case stderrors.Is(err, service.ErrSamePassword):
			appErr := errors.NewValidationError("New password must differ from current password").
				WithDetails(service.DetailSamePassword)
			c.Error(appErr)
			return
		default:
			logger.Errorf(ctx, "Failed to change password: %v", err)
			appErr := errors.NewBadRequestError("Password change failed").WithDetails(err.Error())
			c.Error(appErr)
			return
		}
	}

	logger.Infof(ctx, "Password changed successfully for user: %s", user.Email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// GetAuthConfig godoc
// @Summary      Kimlik doğrulama yapılandırmasını al
// @Description  Geçerli dağıtımın kayıt modunu ve parola karmaşıklığı anahtarını döndürür; ön yüzün kayıt girişini gösterip göstermeyeceğini ve parola doğrulama kurallarını belirlemesi için
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Kimlik doğrulama yapılandırması"
// @Router       /auth/config [get]
//
// GetAuthConfig is intentionally a no-auth endpoint: the frontend reads
// it on app load to decide whether to show the Register tab and which
// password complexity rules to apply. We expose only what the UI
// strictly needs; other config stays internal.
func (h *AuthHandler) GetAuthConfig(c *gin.Context) {
	// Same source-of-truth as Register's gate, so the UI hide-the-button
	// signal can never disagree with the API enforcement signal.
	mode := h.resolveRegistrationMode(c.Request.Context())

	complexPasswordEnabled := service.ResolveComplexPasswordEnabled(
		c.Request.Context(),
		h.configInfo,
		h.systemSettingSvc,
	)
	c.JSON(http.StatusOK, gin.H{
		"success":                  true,
		"registration_mode":        mode,
		"complex_password_enabled": complexPasswordEnabled,
	})
}

// SwitchTenant godoc
// @Summary      Etkin alanı değiştir
// @Description  Geçerli kullanıcı için hedef alanda erişim belirtecini yeniden düzenler; kullanıcının hedef alanda active üyelik ilişkisi olması gerekir (kiracılar arası süper kullanıcılar hariç).
// @Description  Başarılı belirteç yenileme, hedef alanı 「son etkin kiracı」 tercihine yazar; sonraki giriş ve refresh bu alanda gerçekleşir (refresh JWT, tenant_id içermez).
// @Description  Bu tercih hesap düzeyindedir: tek bir belirteç yenileme, bu kullanıcının tüm cihazlarındaki sonraki giriş/refresh hedefini değiştirir. Tercih yazılamazsa tüm belirteç yenileme işlemi başarısız olur ve yeni token verilmez.
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Param        request  body      object{tenant_id=integer,refresh_token=string}  true  "Değiştirme isteği"
// @Success      200      {object}  types.LoginResponse
// @Failure      400      {object}  errors.AppError  "Parametre hatası"
// @Failure      403      {object}  errors.AppError  "Bu alan için üyelik ilişkisi yok veya tercih yazılamadı"
// @Security     Bearer
// @Router       /auth/switch-tenant [post]
//
// SwitchTenant is the v1 backend hook for the tenant-switcher UI added
// in PR 3. The current PR ships the endpoint so multi-tenant tests can
// exercise the membership flow end-to-end before the frontend lands.
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		TenantID     uint64 `json:"tenant_id"     binding:"required"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := errors.NewValidationError("Invalid workspace switch request").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil || user == nil {
		appErr := errors.NewUnauthorizedError("not authenticated")
		c.Error(appErr)
		return
	}

	resp, err := h.userService.SwitchTenant(ctx, user, req.TenantID, req.RefreshToken)
	if err != nil {
		logger.Errorf(ctx, "SwitchTenant failed user=%s target=%d: %v", user.ID, req.TenantID, err)
		appErr := errors.NewForbiddenError("workspace switch failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	c.JSON(http.StatusOK, dto.NewAuthLoginResponse(resp))
}

// tenantNameOrEmpty returns t.Name when t is non-nil, "" otherwise.
// Used when building invitation membership responses.
func tenantNameOrEmpty(t *types.Tenant) string {
	if t == nil {
		return ""
	}
	return t.Name
}

// Heartbeat godoc
// @Summary      Presence heartbeat
// @Description  Keeps the caller marked as online in the system-admin users table.
// @Description  Presence itself is recorded by the auth middleware; this endpoint only
// @Description  gives an open but idle tab a cheap request to make.
// @Tags         Kimlik Doğrulama
// @Success      204  "Recorded"
// @Security     Bearer
// @Router       /auth/heartbeat [post]
func (h *AuthHandler) Heartbeat(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ValidateToken godoc
// @Summary      Belirteci doğrula
// @Description  Erişim belirtecinin geçerli olup olmadığını doğrular
// @Tags         Kimlik Doğrulama
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Belirteç geçerli"
// @Failure      401  {object}  errors.AppError         "Belirteç geçersiz"
// @Security     Bearer
// @Router       /auth/validate [get]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start token validation")

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger.Error(ctx, "Missing Authorization header")
		appErr := errors.NewValidationError("Authorization header is required")
		c.Error(appErr)
		return
	}

	// Parse Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger.Error(ctx, "Invalid Authorization header format")
		appErr := errors.NewValidationError("Invalid Authorization header format")
		c.Error(appErr)
		return
	}

	token := tokenParts[1]

	// Validate token
	user, _, err := h.userService.ValidateToken(ctx, token)
	if err != nil {
		logger.Errorf(ctx, "Failed to validate token: %v", err)
		appErr := errors.NewUnauthorizedError("Token validation failed").WithDetails(err.Error())
		c.Error(appErr)
		return
	}

	logger.Infof(ctx, "Token validated successfully for user: %s", user.Email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token is valid",
		"user":    user.ToUserInfo(),
	})
}
