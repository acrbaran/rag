package repository

import (
	"context"
	"testing"
	"time"

	"github.com/acrbaran/rag/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositoryUpdatePresenceSurvivesStaleSave(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_presence?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&types.Tenant{}, &types.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	repo := NewUserRepository(db)
	user := &types.User{ID: "presence-user", Username: "presence", Email: "presence@example.com", PasswordHash: "x", IsActive: true}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	stale, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if stale.LastSeenAt != nil || stale.OnlineSince != nil {
		t.Fatalf("new user must have no presence, got %+v / %+v", stale.LastSeenAt, stale.OnlineSince)
	}

	since := time.Now().Add(-10 * time.Minute).UTC().Truncate(time.Second)
	seen := time.Now().UTC().Truncate(time.Second)
	if err := repo.UpdatePresence(ctx, user.ID, seen, &since); err != nil {
		t.Fatalf("UpdatePresence: %v", err)
	}

	// A Save of the copy loaded before the presence write must not roll the
	// presence columns back.
	stale.Username = "renamed"
	if err := repo.UpdateUser(ctx, stale); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Username != "renamed" {
		t.Fatalf("username = %q, want renamed", got.Username)
	}
	if got.LastSeenAt == nil || !got.LastSeenAt.Equal(seen) {
		t.Fatalf("last_seen_at = %v, want %v", got.LastSeenAt, seen)
	}
	if got.OnlineSince == nil || !got.OnlineSince.Equal(since) {
		t.Fatalf("online_since = %v, want %v", got.OnlineSince, since)
	}
	if p := got.PresenceAt(seen.Add(time.Minute)); !p.Online || p.OnlineSince == nil {
		t.Fatalf("presence one minute later = %+v, want online", p)
	}
	if p := got.PresenceAt(seen.Add(types.PresenceOnlineWindow)); p.Online || p.OnlineSince != nil || p.LastSeenAt == nil {
		t.Fatalf("presence after the online window = %+v, want offline with last_seen_at", p)
	}

	// Logout clears the streak.
	if err := repo.UpdatePresence(ctx, user.ID, seen, nil); err != nil {
		t.Fatalf("UpdatePresence(offline): %v", err)
	}
	got, err = repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.OnlineSince != nil || got.PresenceAt(seen).Online {
		t.Fatalf("user must be offline after clearing, got online_since=%v", got.OnlineSince)
	}
}
