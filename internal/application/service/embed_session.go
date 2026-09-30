package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/utils"
	"github.com/redis/go-redis/v9"
)

const (
	embedSessionTokenPrefix = "ems_"
	embedSessionRedisPrefix = "embed:session:"
	embedSessionTTL         = 30 * time.Minute
)

var ErrEmbedSessionUnavailable = errors.New("embed session tokens unavailable")

func generateEmbedSessionToken() (string, error) {
	buf := make([]byte, embedTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return embedSessionTokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// embedPublishTokenFingerprint returns a short, non-reversible fingerprint of the
// channel's current publish token. Session tokens store it so that rotating the
// publish token immediately invalidates every outstanding session token.
func embedPublishTokenFingerprint(publishToken string) string {
	sum := sha256.Sum256([]byte("embed-session-fp:v1|" + publishToken))
	return hex.EncodeToString(sum[:8])
}

// IssueSessionToken mints a short-lived session token bound to channelID.
func (s *embedChannelService) IssueSessionToken(ctx context.Context, channelID string) (string, int, error) {
	if s.redis == nil {
		return "", 0, ErrEmbedSessionUnavailable
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", 0, ErrEmbedTokenInvalid
	}
	ch, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return "", 0, err
	}
	if ch == nil {
		return "", 0, ErrEmbedTokenInvalid
	}
	token, err := generateEmbedSessionToken()
	if err != nil {
		return "", 0, err
	}
	key := embedSessionRedisPrefix + token
	value := ch.ID + ":" + embedPublishTokenFingerprint(ch.PublishToken)
	if err := s.redis.Set(ctx, key, value, embedSessionTTL).Err(); err != nil {
		return "", 0, err
	}
	return token, int(embedSessionTTL.Seconds()), nil
}

// ResolveSessionToken returns the channel ID stored for a session token. The
// token is rejected when the channel's publish token was rotated after issue.
func (s *embedChannelService) ResolveSessionToken(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, embedSessionTokenPrefix) {
		return "", ErrEmbedTokenInvalid
	}
	if s.redis == nil {
		return "", ErrEmbedSessionUnavailable
	}
	key := embedSessionRedisPrefix + token
	value, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrEmbedTokenInvalid
	}
	if err != nil {
		return "", err
	}
	channelID, fingerprint, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || channelID == "" || fingerprint == "" {
		// Legacy value without a publish-token fingerprint: force re-exchange.
		return "", ErrEmbedTokenInvalid
	}
	ch, err := s.repo.GetByID(ctx, channelID)
	if err != nil || ch == nil {
		return "", ErrEmbedTokenInvalid
	}
	expected := embedPublishTokenFingerprint(ch.PublishToken)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(fingerprint)) != 1 {
		_ = s.redis.Del(ctx, key).Err()
		return "", ErrEmbedTokenInvalid
	}
	return channelID, nil
}

// LookupEnabledChannel loads an embed channel and verifies it is enabled.
func (s *embedChannelService) LookupEnabledChannel(ctx context.Context, channelID string) (*types.EmbedChannel, error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, ErrEmbedTokenInvalid
	}
	ch, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrEmbedTokenInvalid
	}
	if !ch.Enabled {
		return nil, ErrEmbedChannelDisabled
	}
	return ch, nil
}

// IsEmbedSessionToken reports whether token is a session token (ems_ prefix).
func IsEmbedSessionToken(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), embedSessionTokenPrefix)
}

// SignEmbedSessionHandle binds a chat session id to its embed channel with an
// HMAC keyed by a server-only deployment secret. The handle is handed
// to the widget at session-creation time and must be presented on every history
// load / chat call. Because the session id travels in the request path (and can
// land in access logs), this signature — sent in a header, never logged — is the
// real authorization secret: a leaked session id is useless without it. Rotating
// the channel token invalidates outstanding handles, which is acceptable.
func SignEmbedSessionHandle(ch *types.EmbedChannel, sessionID string) string {
	if ch == nil || strings.TrimSpace(sessionID) == "" {
		return ""
	}
	key := utils.SystemHMACKey()
	if len(key) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("embed-session:v2|" + ch.PublishToken + "|" + ch.ID + "|" + sessionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyEmbedSessionHandle reports whether sig is a valid handle for sessionID
// on channel ch, using a constant-time comparison.
func VerifyEmbedSessionHandle(ch *types.EmbedChannel, sessionID, sig string) bool {
	sig = strings.TrimSpace(sig)
	if sig == "" {
		return false
	}
	expected := SignEmbedSessionHandle(ch, sessionID)
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) == 1
}

// IssuePreviewSession mints a short-lived session token for management UI preview.
func (s *embedChannelService) IssuePreviewSession(
	ctx context.Context, tenantID uint64, channelID string,
) (string, int, error) {
	ch, err := s.getOwned(ctx, tenantID, channelID)
	if err != nil {
		return "", 0, err
	}
	if !ch.Enabled {
		return "", 0, ErrEmbedChannelDisabled
	}
	return s.IssueSessionToken(ctx, ch.ID)
}
