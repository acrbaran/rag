package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/acrbaran/rag/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositorySetSystemRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_system_role?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&types.Tenant{}, &types.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	repo := NewUserRepository(db)
	for _, u := range []*types.User{
		{ID: "role-actor", Username: "actor", Email: "actor@example.com", PasswordHash: "x", IsActive: true, IsSystemAdmin: true},
		{ID: "role-target", Username: "target", Email: "target@example.com", PasswordHash: "x", IsActive: true},
	} {
		if err := repo.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", u.ID, err)
		}
	}

	set := func(role types.SystemRole) (*types.User, types.SystemRole, bool, error) {
		return repo.SetSystemRole(ctx, "role-target", "role-actor", role)
	}

	if _, _, _, err := set("owner"); !errors.Is(err, ErrInvalidSystemRole) {
		t.Fatalf("invalid role: err = %v, want ErrInvalidSystemRole", err)
	}
	if _, _, _, err := repo.SetSystemRole(ctx, "role-actor", "role-actor", types.SystemRoleUser); !errors.Is(err, ErrCannotChangeOwnRole) {
		t.Fatalf("self change: err = %v, want ErrCannotChangeOwnRole", err)
	}
	if _, _, _, err := repo.SetSystemRole(ctx, "missing", "role-actor", types.SystemRoleUser); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing user: err = %v, want ErrUserNotFound", err)
	}

	steps := []struct {
		role        types.SystemRole
		wantFrom    types.SystemRole
		wantChanged bool
		admin, all  bool
	}{
		{types.SystemRoleWorkspaceAdmin, types.SystemRoleUser, true, false, true},
		{types.SystemRoleWorkspaceAdmin, types.SystemRoleWorkspaceAdmin, false, false, true},
		{types.SystemRoleSuperAdmin, types.SystemRoleWorkspaceAdmin, true, true, true},
		{types.SystemRoleSystemAdmin, types.SystemRoleSuperAdmin, true, true, false},
		{types.SystemRoleUser, types.SystemRoleSystemAdmin, true, false, false},
	}
	for _, step := range steps {
		user, from, changed, err := set(step.role)
		if err != nil {
			t.Fatalf("set %s: %v", step.role, err)
		}
		if from != step.wantFrom || changed != step.wantChanged {
			t.Fatalf("set %s: from=%s changed=%v, want from=%s changed=%v",
				step.role, from, changed, step.wantFrom, step.wantChanged)
		}
		got, err := repo.GetUserByID(ctx, "role-target")
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.IsSystemAdmin != step.admin || got.CanAccessAllTenants != step.all || user.SystemRole() != step.role {
			t.Fatalf("after %s: is_system_admin=%v can_access_all_tenants=%v", step.role, got.IsSystemAdmin, got.CanAccessAllTenants)
		}
	}

	// role-actor is now the only system admin; demoting it must be refused.
	if _, _, _, err := repo.SetSystemRole(ctx, "role-actor", "role-target", types.SystemRoleWorkspaceAdmin); !errors.Is(err, ErrLastSystemAdmin) {
		t.Fatalf("last admin: err = %v, want ErrLastSystemAdmin", err)
	}
}
