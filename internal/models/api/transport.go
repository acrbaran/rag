package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	secutils "github.com/acrbaran/rag/internal/utils"
)

// LLM çağrısı zaman aşımı yapılandırması. Yalnızca "üst katmanda deadline ayarlanmadığında" yedek olarak kullanılır; takılmış isteklerin
// worker'ı kalıcı olarak engellemesini önler. Üst ctx zaten bir deadline ayarladıysa (varsayılandan daha kısa veya daha uzun olması fark etmeksizin),
// aynen buna uyulur; varsayılan zaman aşımı tekrar eklenmez. Ortam değişkenleriyle geçersiz kılınabilir:
//   - RETHRA_LLM_CHAT_TIMEOUT_SECONDS    Akışsız çağrılar için yedek zaman aşımı (varsayılan 600s)
//   - RETHRA_LLM_STREAM_TIMEOUT_SECONDS  Akışlı çağrılar için yedek zaman aşımı (varsayılan 1800s)
var (
	DefaultChatTimeout   = envDurationSeconds("RETHRA_LLM_CHAT_TIMEOUT_SECONDS", 300*time.Second)
	DefaultStreamTimeout = envDurationSeconds("RETHRA_LLM_STREAM_TIMEOUT_SECONDS", 600*time.Second)
)

// envDurationSeconds, "saniye" birimindeki ortam değişkenini okur; ayrıştırma başarısız olursa veya değer pozitif değilse fallback değerine döner.
func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// WithLLMTimeout, yalnızca üst ctx'de deadline olmadığında yedek bir zaman aşımı ekler;
// üst katman açıkça bir deadline ayarladıysa (daha kısa veya daha uzun olması fark etmeksizin), aynen döner,
// böylece çağıran kendi zaman aşımı stratejisi üzerinde nihai karar yetkisine sahip olur.
func WithLLMTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// HTTPClient is a shared HTTP client for raw HTTP LLM calls with connection-level timeouts.
// Per-request timeout is enforced via context deadline (see DefaultChatTimeout / DefaultStreamTimeout)
// rather than http.Client.Timeout, so streaming calls are not prematurely terminated.
// Uses SSRFSafeDialContext to prevent DNS rebinding attacks at the connection layer.
var httpTransport = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	DialContext:         secutils.SSRFSafeDialContext,
	TLSHandshakeTimeout: 10 * time.Second,
	IdleConnTimeout:     90 * time.Second,
	MaxIdleConnsPerHost: 5,
}

// HTTPClient is the shared SSRF-safe client every protocol uses.
var HTTPClient = secutils.NewSSRFSafeHTTPClientWithTransport(
	secutils.SSRFSafeHTTPClientConfig{Timeout: 0, MaxRedirects: 10},
	httpTransport,
)
