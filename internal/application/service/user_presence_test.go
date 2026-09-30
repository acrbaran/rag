package service

import (
	"context"
	"testing"
	"time"

	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
)

type presenceWrite struct {
	lastSeenAt  time.Time
	onlineSince *time.Time
}

type presenceUserRepo struct {
	interfaces.UserRepository
	writes []presenceWrite
}

func (r *presenceUserRepo) UpdatePresence(_ context.Context, _ string, lastSeenAt time.Time, onlineSince *time.Time) error {
	r.writes = append(r.writes, presenceWrite{lastSeenAt: lastSeenAt, onlineSince: onlineSince})
	return nil
}

func TestTouchPresence(t *testing.T) {
	ptr := func(v time.Time) *time.Time { return &v }
	now := time.Now()

	t.Run("first sighting starts a streak", func(t *testing.T) {
		repo := &presenceUserRepo{}
		svc := &userService{userRepo: repo}
		user := &types.User{ID: "u"}
		svc.TouchPresence(context.Background(), user)
		if len(repo.writes) != 1 || repo.writes[0].onlineSince == nil {
			t.Fatalf("writes = %+v, want one write with online_since", repo.writes)
		}
		if !user.IsOnlineAt(time.Now()) {
			t.Fatal("user must be online after TouchPresence")
		}
	})

	t.Run("recent activity is throttled", func(t *testing.T) {
		repo := &presenceUserRepo{}
		svc := &userService{userRepo: repo}
		svc.TouchPresence(context.Background(), &types.User{
			ID: "u", LastSeenAt: ptr(now.Add(-10 * time.Second)), OnlineSince: ptr(now.Add(-time.Hour)),
		})
		if len(repo.writes) != 0 {
			t.Fatalf("writes = %+v, want none inside the touch interval", repo.writes)
		}
	})

	t.Run("continuous activity keeps the streak start", func(t *testing.T) {
		repo := &presenceUserRepo{}
		svc := &userService{userRepo: repo}
		since := now.Add(-time.Hour)
		svc.TouchPresence(context.Background(), &types.User{
			ID: "u", LastSeenAt: ptr(now.Add(-time.Minute)), OnlineSince: ptr(since),
		})
		if len(repo.writes) != 1 || repo.writes[0].onlineSince == nil || !repo.writes[0].onlineSince.Equal(since) {
			t.Fatalf("writes = %+v, want online_since kept at %v", repo.writes, since)
		}
	})

	t.Run("returning after the online window restarts the streak", func(t *testing.T) {
		repo := &presenceUserRepo{}
		svc := &userService{userRepo: repo}
		since := now.Add(-time.Hour)
		svc.TouchPresence(context.Background(), &types.User{
			ID: "u", LastSeenAt: ptr(now.Add(-types.PresenceOnlineWindow - time.Minute)), OnlineSince: ptr(since),
		})
		if len(repo.writes) != 1 || repo.writes[0].onlineSince == nil || repo.writes[0].onlineSince.Equal(since) {
			t.Fatalf("writes = %+v, want a fresh online_since", repo.writes)
		}
	})

	t.Run("logged-out user comes back online", func(t *testing.T) {
		repo := &presenceUserRepo{}
		svc := &userService{userRepo: repo}
		svc.TouchPresence(context.Background(), &types.User{ID: "u", LastSeenAt: ptr(now.Add(-5 * time.Second))})
		if len(repo.writes) != 1 || repo.writes[0].onlineSince == nil {
			t.Fatalf("writes = %+v, want a new streak after logout", repo.writes)
		}
	})
}
