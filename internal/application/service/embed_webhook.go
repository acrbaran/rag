package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	secutils "github.com/acrbaran/rag/internal/utils"
)

const embedWebhookTimeout = 5 * time.Second

// ErrEmbedWebhookURLInvalid is returned when a webhook URL fails format or SSRF checks.
var ErrEmbedWebhookURLInvalid = errors.New("invalid embed webhook URL")

func newEmbedWebhookHTTPClient() *http.Client {
	return secutils.NewSSRFSafeHTTPClient(secutils.SSRFSafeHTTPClientConfig{
		Timeout:      embedWebhookTimeout,
		MaxRedirects: 5,
	})
}

// ValidateEmbedWebhookURL checks an optional outbound webhook URL. Empty is allowed.
func ValidateEmbedWebhookURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%w: webhook URL must be a valid https URL", ErrEmbedWebhookURLInvalid)
	}
	// Webhook bodies carry chat content and an HMAC signature; plain http
	// would expose both in transit, so only https is accepted.
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("%w: webhook URL must use https", ErrEmbedWebhookURLInvalid)
	}
	if err := secutils.ValidateURLForSSRF(trimmed); err != nil {
		if hint := secutils.FormatSSRFError("Webhook URL", trimmed, err, "en-US"); hint != "" {
			return fmt.Errorf("%w: %s", ErrEmbedWebhookURLInvalid, hint)
		}
		return fmt.Errorf("%w: %v", ErrEmbedWebhookURLInvalid, err)
	}
	return nil
}

// DispatchEmbedWebhook POSTs an event to the channel webhook URL (best-effort, async).
func DispatchEmbedWebhook(ch *types.EmbedChannel, eventType, sessionID string, payload map[string]any) {
	if ch == nil {
		return
	}
	url := strings.TrimSpace(ch.WebhookURL)
	if url == "" {
		return
	}
	if err := ValidateEmbedWebhookURL(url); err != nil {
		logger.Warnf(context.Background(), "[embed_webhook] skip dispatch %s: %v", eventType, err)
		return
	}
	body := map[string]any{
		"type":       eventType,
		"channel_id": ch.ID,
		"session_id": sessionID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range payload {
		body[k] = v
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	secret := strings.TrimSpace(ch.WebhookSecret)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), embedWebhookTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Rethra-Embed-Webhook/1.0")
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			_, _ = mac.Write(raw)
			req.Header.Set("X-Rethra-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
		resp, err := newEmbedWebhookHTTPClient().Do(req)
		if err != nil {
			logger.Warnf(context.Background(), "[embed_webhook] dispatch %s failed: %v", eventType, err)
			return
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 300 {
			logger.Warnf(context.Background(), "[embed_webhook] dispatch %s HTTP %d", eventType, resp.StatusCode)
		}
	}()
}

// SignEmbedWebhookBody returns the hex HMAC signature for tests.
func SignEmbedWebhookBody(secret string, raw []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}
