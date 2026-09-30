package handler

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/application/service"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/sandbox"
	"github.com/acrbaran/rag/internal/types"
)

// --- Sandbox connectivity check ---

// SandboxCheckRequest is the body for POST /system/sandbox-check. Secrets may
// arrive redacted; they are resolved against the workspace's stored config so
// an admin can test without retyping an API key.
type SandboxCheckRequest struct {
	Config *types.TenantSandboxConfig `json:"config"`
	// ConfigID lets an edit form test stored credentials while overriding only
	// the fields the admin changed in the drawer.
	ConfigID string `json:"config_id"`
	// Deep additionally runs a throwaway script. For remote backends this also
	// creates and destroys one sandbox, which is the only way to validate the
	// template ID, data plane, in-sandbox execution, and outbound egress. It may
	// consume real sandbox time, so it is opt-in.
	Deep bool `json:"deep"`
}

// SandboxCheckItem is one probe outcome. OK is nil when the probe was not
// executed, which distinguishes "skipped" from "failed" in the UI.
type SandboxCheckItem struct {
	Name string `json:"name"`
	OK   *bool  `json:"ok"`
	// Message carries free-form provider detail for an executed probe.
	Message string `json:"message,omitempty"`
	// Reason is a stable code explaining why a probe was skipped. It exists so
	// the UI can phrase the skip in the operator's language instead of echoing
	// a server-side sentence.
	Reason    string `json:"reason,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
}

// Why a probe was not executed. Kept as codes because the operator reads them
// in the frontend, which is localized.
const (
	// The probe needs a real sandbox, so it only runs on an opt-in deep check.
	skipReasonNeedsDeepCheck = "needs_deep_check"
	// An earlier probe failed and this one cannot be reached without it.
	skipReasonControlPlaneUnreachable = "control_plane_unreachable"
	skipReasonSandboxNotCreated       = "sandbox_not_created"
	skipReasonSandboxExecFailed       = "sandbox_exec_failed"
	// The config denies egress by default, so a probe that cannot reach the
	// internet is the policy working as configured, not a fault.
	skipReasonEgressRestrictedByPolicy = "egress_restricted_by_policy"
)

// SandboxCheckResponse aggregates the probes for one sandbox configuration.
type SandboxCheckResponse struct {
	OK           bool               `json:"ok"`
	Provider     string             `json:"provider"`
	Checks       []SandboxCheckItem `json:"checks"`
	Capabilities map[string]bool    `json:"capabilities,omitempty"`
}

// add records an executed probe. A single failure fails the whole result.
func (r *SandboxCheckResponse) add(name string, ok bool, message string, latencyMS int64) {
	value := ok
	r.Checks = append(r.Checks, SandboxCheckItem{
		Name: name, OK: &value, Message: message, LatencyMS: latencyMS,
	})
	if !ok {
		r.OK = false
	}
}

// skip records a probe that was not run; it never affects OK.
func (r *SandboxCheckResponse) skip(name, reason string) {
	r.Checks = append(r.Checks, SandboxCheckItem{Name: name, OK: nil, Reason: reason})
}

// CheckSandboxConfig tests a sandbox configuration without persisting it.
// @Summary      Sandbox bağlantısını test et
// @Description  Mevcut girilen parametrelerle sandbox arka ucunu test eder, yapılandırmayı kaydetmez; deep=true geçici bir betik çalıştırır, uzak arka uç ayrıca bir sandbox oluşturup yok eder
// @Tags         Sistem
// @Accept       json
// @Produce      json
// @Param        body  body  SandboxCheckRequest  true  "Sandbox yapılandırması"
// @Success      200   {object}  SandboxCheckResponse
// @Router       /system/sandbox-check [post]
func (h *SystemHandler) CheckSandboxConfig(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())

	var req SandboxCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": types.LocalizedText(ctx, "İstek gövdesinin biçimi geçersiz", "Invalid request body format")})
		return
	}
	tenant, _ := types.TenantInfoFromContext(c.Request.Context())
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": types.LocalizedText(ctx, "Çalışma alanı bulunamadı", "Workspace not found")})
		return
	}

	var stored *types.TenantSandboxConfig
	incoming := req.Config
	if req.ConfigID != "" {
		if h.sandboxConfigSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": types.LocalizedText(ctx, "Sandbox yapılandırma hizmeti kullanılamıyor", "Sandbox configuration service is unavailable")})
			return
		}
		entity, err := h.sandboxConfigSvc.Get(ctx, tenant.ID, req.ConfigID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
			return
		}
		if entity == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": types.LocalizedText(ctx, "Sandbox yapılandırması bulunamadı", "Sandbox configuration not found")})
			return
		}
		stored = entity.Config
		if incoming == nil {
			incoming = stored
		}
	}
	if !req.Deep {
		incoming = sandboxConnectionCheckConfig(incoming)
	}
	merged, err := service.SanitizeSandboxConfig(incoming, stored)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	effective, err := sandbox.ResolveEffectiveConfig(merged, sandbox.DefaultConfig())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	result := &SandboxCheckResponse{OK: true, Provider: string(effective.Type)}
	client, err := sandbox.NewRemoteClientForCheck(effective)
	if err != nil {
		result.add("client_build", false, err.Error(), 0)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
		return
	}

	// Level 1: an authenticated control-plane call, which validates endpoint
	// reachability AND credentials in a single round-trip.
	start := time.Now()
	healthErr := client.Health(ctx)
	latency := time.Since(start).Milliseconds()
	if healthErr != nil {
		result.add("api_url_reachable", false, sandboxCheckReason(ctx, healthErr), latency)
		result.skip("credential_valid", skipReasonControlPlaneUnreachable)
	} else {
		result.add("api_url_reachable", true, "", latency)
		result.add("credential_valid", true, "", 0)
	}

	caps := client.Capabilities()
	result.Capabilities = map[string]bool{
		"supports_volumes":      caps.SupportsVolumes,
		"supports_pause_resume": caps.SupportsPauseResume,
		"supports_reconnect":    caps.SupportsReconnect,
	}

	if !req.Deep || healthErr != nil {
		reason := skipReasonNeedsDeepCheck
		if healthErr != nil {
			reason = skipReasonControlPlaneUnreachable
		}
		result.skip("template_exists", reason)
		result.skip("sandbox_exec", reason)
		result.skip("egress_available", reason)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
		return
	}

	h.runDeepSandboxCheck(ctx, client, effective, result)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// sandboxConnectionCheckConfig supplies a private template placeholder for a
// shallow remote probe. Connectivity is deliberately checked before template
// discovery in the settings wizard, and Health never uses this value.
func sandboxConnectionCheckConfig(cfg *types.TenantSandboxConfig) *types.TenantSandboxConfig {
	if cfg == nil {
		return nil
	}
	copy := *cfg
	switch sandbox.SandboxType(copy.SandboxType) {
	case sandbox.SandboxTypeCube:
		cube := types.CubeSandboxConfig{}
		if copy.Cube != nil {
			cube = *copy.Cube
		}
		if strings.TrimSpace(cube.TemplateID) == "" {
			cube.TemplateID = "__connection_check__"
		}
		copy.Cube = &cube
	case sandbox.SandboxTypeE2B:
		e2b := types.E2BSandboxConfig{}
		if copy.E2B != nil {
			e2b = *copy.E2B
		}
		if strings.TrimSpace(e2b.TemplateID) == "" {
			e2b.TemplateID = "__connection_check__"
		}
		copy.E2B = &e2b
	}
	return &copy
}

// describeProbeMismatch reports why the probe script did not print its marker.
// "Komut çıktısı beklendiği gibi değil" tek başına çıkış kodunu ve stderr satırını gizler
// normally name the actual problem — a missing interpreter, an image entrypoint
// that swallowed the command, or a bind mount that never reached the Docker VM.
func describeProbeMismatch(ctx context.Context, exitCode int, killed bool, stdout, stderr, execErr string) string {
	parts := []string{fmt.Sprintf(types.LocalizedText(ctx, "Çıkış kodu %d", "Exit code %d"), exitCode)}
	if killed {
		parts = append(parts, types.LocalizedText(ctx, "Zaman aşımı nedeniyle sonlandırıldı", "Terminated after timeout"))
	}
	switch {
	case firstProbeLine(stderr) != "":
		parts = append(parts, "stderr: "+firstProbeLine(stderr))
	case firstProbeLine(stdout) != "":
		parts = append(parts, "stdout: "+firstProbeLine(stdout))
	default:
		parts = append(parts, types.LocalizedText(ctx, "Çıktı yok", "No output"))
	}
	if trimmed := strings.TrimSpace(execErr); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "; ")
}

// firstProbeLine picks the first non-empty line and caps it, so one runaway log
// line cannot flood the check panel.
func firstProbeLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 300 {
			return trimmed[:300] + "…"
		}
		return trimmed
	}
	return ""
}

// runDeepSandboxCheck creates one throwaway sandbox and verifies a command can
// run inside it. The sandbox is always deleted, including on failure.
func (h *SystemHandler) runDeepSandboxCheck(
	ctx context.Context,
	client sandbox.RemoteSandboxClient,
	cfg *sandbox.Config,
	result *SandboxCheckResponse,
) {
	probeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	handle, err := client.Create(probeCtx, sandbox.RemoteCreateRequest{
		TemplateID: sandbox.EffectiveTemplateID(cfg),
		// Probe under the admin's policy: the point of this check is whether
		// THIS config works, not whether the provider's default does.
		Network: cfg.Network,
		Timeout: sandbox.RemoteTimeoutPolicy{
			Mode:   sandbox.RemoteTimeoutExplicit,
			Value:  2 * time.Minute,
			Action: sandbox.RemoteOnTimeoutKill,
		},
	})
	if err != nil {
		result.add("template_exists", false, explainSandboxCreateFailure(ctx, client, cfg, err), 0)
		result.skip("sandbox_exec", skipReasonSandboxNotCreated)
		result.skip("egress_available", skipReasonSandboxNotCreated)
		return
	}
	defer func() {
		// Detach from ctx so cleanup still runs if the request was cancelled;
		// a leaked probe sandbox would otherwise sit there billing.
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.WithoutCancel(ctx), 30*time.Second)
		defer cleanupCancel()
		if err := client.Delete(cleanupCtx, handle.ID()); err != nil {
			logger.Warnf(ctx, "[SandboxCheck] failed to delete probe sandbox %s: %v",
				handle.ID(), err)
		}
	}()
	result.add("template_exists", true, "", 0)
	logger.Infof(ctx,
		"[SandboxCheck] probe sandbox created id=%s provider=%s template=%s",
		handle.ID(), result.Provider, sandbox.EffectiveTemplateID(cfg),
	)

	const marker = "rethra-ok"
	start := time.Now()
	execResult, err := client.Exec(probeCtx, handle, sandbox.RemoteExecRequest{
		Command: "echo",
		Args:    []string{marker},
		User:    sandbox.DefaultSandboxExecUser,
		Timeout: 30 * time.Second,
	})
	latency := time.Since(start).Milliseconds()
	switch {
	case err != nil:
		logger.Warnf(ctx,
			"[SandboxCheck] sandbox_exec failed sandbox=%s latency_ms=%d ui_reason=%s detail=%s",
			handle.ID(), latency, sandboxCheckReason(ctx, err), sandbox.RemoteErrorDiagnostics(err),
		)
		result.add("sandbox_exec", false, sandboxCheckReason(ctx, err), latency)
		result.skip("egress_available", skipReasonSandboxExecFailed)
		return
	case execResult == nil:
		result.add("sandbox_exec", false, types.LocalizedText(ctx, "Sandbox çalıştırma sonucu döndürmedi", "Sandbox did not return an execution result"), latency)
		result.skip("egress_available", skipReasonSandboxExecFailed)
		return
	case !strings.Contains(execResult.Stdout, marker):
		logger.Warnf(ctx,
			"[SandboxCheck] sandbox_exec probe mismatch sandbox=%s exit=%d killed=%v stdout=%q stderr=%q",
			handle.ID(), execResult.ExitCode, execResult.Killed,
			firstProbeLine(execResult.Stdout), firstProbeLine(execResult.Stderr),
		)
		result.add("sandbox_exec", false, describeProbeMismatch(ctx,
			execResult.ExitCode, execResult.Killed,
			execResult.Stdout, execResult.Stderr, "",
		), latency)
		result.skip("egress_available", skipReasonSandboxExecFailed)
		return
	default:
		result.add("sandbox_exec", true, "", latency)
	}

	h.probeSandboxEgress(probeCtx, client, handle, cfg.Network, result)
}

// egressProbeTargets are tried in order; the first reachable one passes
// egress_available. Domestic and international endpoints cover regional
// egress policies without requiring both to succeed.
var egressProbeTargets = []struct {
	label string
	url   string
}{
	{label: "cn:baidu", url: "https://www.baidu.com"},
	{label: "intl:1.1.1.1", url: "https://1.1.1.1"},
}

// probeSandboxEgress verifies the sandbox can reach the public internet.
// Any single target succeeding is enough — skill installs only need some
// outbound path, not both CN and international reachability.
func (h *SystemHandler) probeSandboxEgress(
	ctx context.Context,
	client sandbox.RemoteSandboxClient,
	handle sandbox.RemoteSandboxHandle,
	policy sandbox.RemoteNetworkPolicy,
	result *SandboxCheckResponse,
) {
	// Echo which target succeeded so the UI message is actionable when
	// only one region is reachable. First success exits 0 immediately.
	var b strings.Builder
	for _, target := range egressProbeTargets {
		fmt.Fprintf(&b,
			`if curl -fsS -o /dev/null -m 8 -I %s; then echo %s; exit 0; fi; `,
			shellSingleQuote(target.url), shellSingleQuote(target.label))
	}
	b.WriteString(`echo "all probes failed" >&2; exit 1`)

	start := time.Now()
	execResult, err := client.Exec(ctx, handle, sandbox.RemoteExecRequest{
		Command: b.String(),
		Shell:   true,
		User:    sandbox.DefaultSandboxExecUser,
		Timeout: 30 * time.Second,
	})
	latency := time.Since(start).Milliseconds()
	switch {
	case err != nil:
		reportEgressProbe(result, policy, false, sandboxCheckReason(ctx, err), latency)
	case execResult == nil:
		reportEgressProbe(result, policy, false, types.LocalizedText(ctx, "Dış ağ denetimi sonuç döndürmedi", "Outbound network probe returned no result"), latency)
	case execResult.Killed:
		reportEgressProbe(result, policy, false, types.LocalizedText(ctx, "Dış ağ denetimi zaman aşımına uğradı", "Outbound network probe timed out"), latency)
	case execResult.ExitCode != 0:
		msg := strings.TrimSpace(execResult.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(execResult.Stdout)
		}
		if msg == "" {
			msg = types.LocalizedText(ctx, "Yerel ve uluslararası denetim hedeflerine ulaşılamıyor", "Neither regional nor international probe target is reachable")
		}
		reportEgressProbe(result, policy, false, msg, latency)
	default:
		hit := strings.TrimSpace(execResult.Stdout)
		if hit == "" {
			hit = "ok"
		}
		reportEgressProbe(result, policy, true, "reachable via "+hit, latency)
	}
}

// reportEgressProbe records the outbound-connectivity probe. Under a
// deny-by-default policy a blocked probe is the expected outcome — the probe
// target is not on the admin's allow list — so it is reported as "restricted
// by policy" instead of failing the whole check. Only a config that allows
// public egress can fail this probe.
func reportEgressProbe(
	result *SandboxCheckResponse,
	policy sandbox.RemoteNetworkPolicy,
	reachable bool,
	detail string,
	latencyMS int64,
) {
	if reachable {
		result.add("egress_available", true, detail, latencyMS)
		return
	}
	if !policy.DeniesEgressByDefault() {
		result.add("egress_available", false, detail, latencyMS)
		return
	}
	result.Checks = append(result.Checks, SandboxCheckItem{
		Name:      "egress_available",
		OK:        nil,
		Reason:    skipReasonEgressRestrictedByPolicy,
		LatencyMS: latencyMS,
	})
}

// explainSandboxCreateFailure turns a failed sandbox creation into a cause the
// operator can act on.
//
// A 404 from the provider is not proof that the template is gone: the E2B SDK
// discards the response body of any 404 on the create endpoint and substitutes
// a fixed "template not found" sentence, so the message alone cannot tell a
// deleted template apart from one whose build cannot boot yet. The catalog is
// the only place that distinction exists, so it is consulted before blaming the
// template ID the admin just picked from a list.
func explainSandboxCreateFailure(
	ctx context.Context,
	client sandbox.RemoteSandboxClient,
	cfg *sandbox.Config,
	err error,
) string {
	reason := sandboxCheckReason(ctx, err)
	var remoteErr *sandbox.RemoteError
	if !stderrors.As(err, &remoteErr) {
		return reason
	}
	// A create-time 404 is classified as InvalidRequest, not NotFound, because
	// for the lifecycle it means "bad argument" rather than "sandbox is gone".
	// Diagnosis needs the status itself.
	if remoteErr.Kind != sandbox.RemoteErrorKindNotFound &&
		remoteErr.StatusCode != http.StatusNotFound {
		return reason
	}
	templateID := strings.TrimSpace(sandbox.EffectiveTemplateID(cfg))
	catalog, ok := client.(sandbox.RemoteTemplateCatalog)
	if !ok || templateID == "" {
		return reason
	}
	templates, listErr := catalog.ListTemplates(ctx)
	if listErr != nil {
		logger.Warnf(ctx, "[SandboxCheck] template lookup after create 404 failed: %v", listErr)
		return reason
	}
	for _, tpl := range templates {
		if tpl.ID != templateID && !strings.EqualFold(tpl.Name, templateID) {
			continue
		}
		if tpl.Status == sandbox.TemplateStatusUntagged {
			return fmt.Sprintf(types.LocalizedText(ctx,
				"%s şablonunun derlemesi tamamlandı ancak hiçbir derlemede %q etiketi yok; sandbox oluşturulamıyor. Şablonu yeniden derleyin (silip Rethra'nın yeniden oluşturmasını sağlayabilirsiniz).",
				"Template %s has finished building, but no build has the %q tag, so a sandbox cannot be created. Rebuild the template (you can delete it and let Rethra recreate it)."),
				templateID, sandbox.DefaultE2BTemplateTag,
			)
		}
		if tpl.Status == "ready" || tpl.Status == "" {
			return fmt.Sprintf(types.LocalizedText(ctx,
				"%s şablonu listede hazır görünüyor ancak küme sandbox oluşturmayı reddetti (HTTP 404). Derleme anlık görüntüsü henüz etkinleşmemiş olabilir; daha sonra yeniden deneyin veya şablonu yeniden derleyin.",
				"Template %s appears ready in the list, but the cluster rejected sandbox creation (HTTP 404). The build snapshot may not be active yet; retry later or rebuild the template."),
				templateID,
			)
		}
		return fmt.Sprintf(types.LocalizedText(ctx,
			"%s şablonu mevcut ancak derleme durumu %s; sandbox henüz başlatılamıyor. Derlemenin tamamlanmasını bekleyin veya derleme günlüklerini kontrol edin.",
			"Template %s exists, but its build status is %s, so the sandbox cannot start yet. Wait for the build to finish or check the build logs."),
			templateID, tpl.Status,
		)
	}
	return fmt.Sprintf(types.LocalizedText(ctx,
		"Geçerli API anahtarı %s şablonunu göremiyor: şablon silinmiş olabilir veya anahtar başka bir ekibe ya da kümeye ait olabilir.",
		"The current API key cannot see template %s: it may have been deleted, or the key may belong to another team or cluster."),
		templateID,
	)
}

// shellSingleQuote wraps s for safe inclusion in a single-quoted shell string.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func dockerUnavailableCheckReason(ctx context.Context, err *sandbox.RemoteError) string {
	host := dockerHostFromUnavailableMessage(err.Message)
	if host == "" {
		detail := firstProbeLine(err.Message)
		if detail == "" {
			return types.LocalizedText(ctx, "Docker hizmetine bağlanılamıyor", "Cannot connect to the Docker daemon")
		}
		return types.LocalizedText(ctx, "Docker hizmetine bağlanılamıyor: ", "Cannot connect to the Docker daemon: ") + detail
	}
	return fmt.Sprintf(types.LocalizedText(ctx,
		"Docker hizmetine %s adresinden bağlanılamıyor. Adresi boş bırakırsanız yerel docker CLI ayarı (DOCKER_HOST veya geçerli docker context) kullanılır. Colima genellikle unix://$HOME/.colima/default/docker.sock kullanır; Rethra bir konteynerde çalışıyorsa soketi uygulama konteynerine bağlayın.",
		"Cannot connect to the Docker daemon at %s. Leave the address blank to use the local docker CLI setting (DOCKER_HOST or the current docker context). Colima usually uses unix://$HOME/.colima/default/docker.sock; if Rethra runs in a container, mount that socket into the app container."), host)
}

func dockerHostFromUnavailableMessage(message string) string {
	const prefix = "Cannot connect to the Docker daemon at "
	rest, ok := strings.CutPrefix(message, prefix)
	if !ok {
		return ""
	}
	if end := strings.IndexAny(rest, " \t"); end > 0 {
		rest = rest[:end]
	}
	return strings.TrimSuffix(strings.TrimSpace(rest), ".")
}

// sandboxCheckReason turns a provider error into a readable cause using the
// adapter's normalized RemoteError.Kind, so the UI never shows raw SDK text.
func sandboxCheckReason(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var remoteErr *sandbox.RemoteError
	if !stderrors.As(err, &remoteErr) {
		return err.Error()
	}
	switch remoteErr.Kind {
	case sandbox.RemoteErrorKindAuthentication:
		return types.LocalizedText(ctx, "Kimlik doğrulama başarısız: API anahtarı geçersiz veya yetkisiz", "Authentication failed: API key is invalid or lacks permission")
	case sandbox.RemoteErrorKindNotFound:
		return types.LocalizedText(ctx, "Kaynak bulunamadı: şablon kimliğini kontrol edin", "Resource not found: check the template ID")
	case sandbox.RemoteErrorKindTimeout:
		return types.LocalizedText(ctx, "İstek zaman aşımına uğradı: uç noktaya ulaşılamıyor veya yanıt çok yavaş", "Request timed out: endpoint is unreachable or responding too slowly")
	case sandbox.RemoteErrorKindUnavailable:
		if remoteErr.Provider == sandbox.SandboxTypeDocker {
			return dockerUnavailableCheckReason(ctx, remoteErr)
		}
		return types.LocalizedText(ctx, "Hizmet kullanılamıyor: uç nokta bağlantıyı reddetti", "Service unavailable: endpoint refused the connection")
	case sandbox.RemoteErrorKindCapacity:
		return types.LocalizedText(ctx, "Kota yetersiz veya hız sınırına ulaşıldı", "Quota exceeded or rate limit reached")
	case sandbox.RemoteErrorKindUnsupported:
		return types.LocalizedText(ctx, "Bu arka uç işlemi desteklemiyor", "This backend does not support the operation")
	case sandbox.RemoteErrorKindInvalidRequest:
		return types.LocalizedText(ctx, "Geçersiz parametre: ", "Invalid parameter: ") + remoteErr.Message
	default:
		return remoteErr.Message
	}
}
