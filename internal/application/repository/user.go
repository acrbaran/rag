package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrTokenNotFound      = errors.New("token not found")
	ErrCannotRevokeSelf   = errors.New("cannot revoke your own system admin privileges")
	ErrLastSystemAdmin    = errors.New("cannot revoke the last remaining system administrator")
	ErrUserNotSystemAdmin = errors.New("user is not a system administrator")
	ErrCannotDeleteSelf   = errors.New("cannot delete your own account")
	// ErrCannotChangeOwnRole rejects a system admin changing their own
	// system role, which could otherwise lock them out mid-request.
	ErrCannotChangeOwnRole = errors.New("cannot change your own system role")
	ErrInvalidSystemRole   = errors.New("invalid system role")
)

// SoleOwnerError is returned by DeleteUserAccount when the target is the
// only active Owner of one or more workspaces; deleting them would leave
// those workspaces without an owner. Workspaces lists the affected ones.
type SoleOwnerError struct {
	Workspaces []types.Membership
}

func (e *SoleOwnerError) Error() string {
	return "user is the sole owner of one or more workspaces"
}

// userRepository implements user repository interface
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) interfaces.UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a user
func (r *userRepository) CreateUser(ctx context.Context, user *types.User) error {
	// users.tenant_id is nullable in both PostgreSQL and SQLite. GORM would
	// otherwise serialise the uint64 zero value as 0, which violates the
	// PostgreSQL FK and loses the distinction between "not provisioned yet"
	// and a real tenant. Omitting the column stores SQL NULL; reads hydrate it
	// back as zero, the domain sentinel used by tenantless auth flows.
	if user != nil && user.TenantID == 0 {
		return r.db.WithContext(ctx).Omit("tenant_id").Create(user).Error
	}
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID gets a user by ID
func (r *userRepository) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUsersByIDs batch-fetches users by id with a single SELECT … WHERE id IN (…)
// and projects the result into a map keyed by user id. Returns an empty
// map for an empty input slice. Missing ids are silently absent from
// the result (consistent with the interface contract used by tenant
// member hydration).
func (r *userRepository) GetUsersByIDs(ctx context.Context, ids []string) (map[string]*types.User, error) {
	out := make(map[string]*types.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []*types.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

// GetUserByEmail gets a user by email
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername gets a user by username
func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByTenantID gets the first user (owner) of a tenant
func (r *userRepository) GetUserByTenantID(ctx context.Context, tenantID uint64) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at ASC").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (r *userRepository) UpdateUser(ctx context.Context, user *types.User) error {
	if user != nil && user.TenantID == 0 {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Preserve Save's all-fields behaviour while keeping the nullable
			// tenant column out of the struct write, then explicitly store NULL.
			// Writing uint64(0) would violate the PostgreSQL tenant FK.
			if err := tx.Omit("tenant_id").Save(user).Error; err != nil {
				return err
			}
			return tx.Model(&types.User{}).
				Where("id = ?", user.ID).
				UpdateColumn("tenant_id", nil).Error
		})
	}
	return r.db.WithContext(ctx).Save(user).Error
}

// DeleteUser deletes a user
func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&types.User{}).Error
}

// ListUsers lists users with pagination
func (r *userRepository) ListUsers(ctx context.Context, offset, limit int) ([]*types.User, error) {
	var users []*types.User
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ListUsersPaged lists every (non-deleted) user for the system-admin
// management table, newest first, optionally filtered by a case-insensitive
// substring match on email, username, first name, last name or phone. Returns the
// page plus the total matching count. limit <= 0 means "no limit".
func (r *userRepository) ListUsersPaged(
	ctx context.Context, search string, offset, limit int,
) ([]*types.User, int64, error) {
	var users []*types.User
	var total int64

	base := r.db.WithContext(ctx).Model(&types.User{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + escapeLikePattern(search) + "%"
		base = base.Where(
			`(LOWER(email) LIKE LOWER(?) ESCAPE ? OR LOWER(username) LIKE LOWER(?) ESCAPE ?`+
				` OR LOWER(first_name) LIKE LOWER(?) ESCAPE ? OR LOWER(last_name) LIKE LOWER(?) ESCAPE ?`+
				` OR phone LIKE ? ESCAPE ?)`,
			like, likeEscapeChar, like, likeEscapeChar, like, likeEscapeChar, like, likeEscapeChar,
			like, likeEscapeChar)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.Order("created_at DESC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdatePresence writes the presence columns only. The fields are read-only
// on the User model (so UpdateUser's Save cannot clobber them with a stale
// copy), hence the raw table update; it also leaves updated_at untouched.
func (r *userRepository) UpdatePresence(
	ctx context.Context, userID string, lastSeenAt time.Time, onlineSince *time.Time,
) error {
	return r.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		UpdateColumns(map[string]any{"last_seen_at": lastSeenAt, "online_since": onlineSince}).Error
}

// DeleteUserAccount removes a user account on behalf of a system admin in a
// single transaction:
//   - rejects deleting the actor's own account (ErrCannotDeleteSelf);
//   - rejects deleting the last remaining system admin (ErrLastSystemAdmin),
//     locking the admin rows like RevokeSystemAdmin so concurrent deletes
//     cannot leave the platform without an administrator;
//   - rewrites username/email to tombstone values so the unique indexes
//     don't block the address from registering again;
//   - soft-deletes every workspace membership and the user row;
//   - revokes every outstanding auth token.
//
// Returns a copy of the user as it was before deletion (original email and
// username) so callers can record it in the audit trail.
func (r *userRepository) DeleteUserAccount(ctx context.Context, userID, actorID string) (*types.User, error) {
	if userID == actorID {
		return nil, ErrCannotDeleteSelf
	}

	var original *types.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}
		var user types.User
		if err := locking(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if user.IsSystemAdmin {
			var admins []types.User
			if err := locking(tx).Where("is_system_admin = ?", true).Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return ErrLastSystemAdmin
			}
		}
		if err := ensureNotSoleOwner(tx, user.ID, locking); err != nil {
			return err
		}
		snapshot := user
		original = &snapshot

		if err := tx.Model(&types.User{}).Where("id = ?", user.ID).UpdateColumns(map[string]any{
			"username":        "deleted_" + user.ID,
			"email":           "deleted_" + user.ID + "@deleted.invalid",
			"is_active":       false,
			"is_system_admin": false,
			"updated_at":      time.Now(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&types.TenantMember{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.AuthToken{}).Where("user_id = ?", user.ID).
			Update("is_revoked", true).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", user.ID).Delete(&types.User{}).Error
	})
	if err != nil {
		return nil, err
	}
	return original, nil
}

// ensureNotSoleOwner returns a *SoleOwnerError when userID is the only
// active Owner of any live workspace. The owner rows of those workspaces are
// locked (postgres/mysql) so a concurrent demote/remove of the co-owner
// cannot slip in between the check and the delete.
func ensureNotSoleOwner(tx *gorm.DB, userID string, locking func(*gorm.DB) *gorm.DB) error {
	var owned []types.TenantMember
	if err := tx.Where("user_id = ? AND role = ? AND status = ?",
		userID, types.TenantRoleOwner, types.TenantMemberStatusActive).
		Find(&owned).Error; err != nil {
		return err
	}
	if len(owned) == 0 {
		return nil
	}
	tenantIDs := make([]uint64, 0, len(owned))
	for _, m := range owned {
		tenantIDs = append(tenantIDs, m.TenantID)
	}

	var owners []types.TenantMember
	if err := locking(tx).Where("tenant_id IN ? AND role = ? AND status = ?",
		tenantIDs, types.TenantRoleOwner, types.TenantMemberStatusActive).
		Find(&owners).Error; err != nil {
		return err
	}
	coOwned := make(map[uint64]bool, len(tenantIDs))
	for _, o := range owners {
		if o.UserID != userID {
			coOwned[o.TenantID] = true
		}
	}
	var sole []uint64
	for _, id := range tenantIDs {
		if !coOwned[id] {
			sole = append(sole, id)
		}
	}
	if len(sole) == 0 {
		return nil
	}

	// Workspaces that were themselves deleted don't need an owner.
	var tenants []types.Tenant
	if err := tx.Select("id", "name").Where("id IN ?", sole).Order("id").Find(&tenants).Error; err != nil {
		return err
	}
	if len(tenants) == 0 {
		return nil
	}
	serr := &SoleOwnerError{Workspaces: make([]types.Membership, 0, len(tenants))}
	for _, t := range tenants {
		serr.Workspaces = append(serr.Workspaces, types.Membership{
			TenantID: t.ID, TenantName: t.Name, Role: types.TenantRoleOwner,
		})
	}
	return serr
}

// ListSystemAdmins lists users where is_system_admin = true.
//
// Walks idx_users_is_system_admin (created in migration 000052), so the
// query stays cheap even on a large users table — only the small subset
// of system admins is scanned. Returns total count alongside the page so
// the management UI can render pagination without a second roundtrip.
//
// Ordered by created_at DESC for stable, newest-first listing; ties are
// further broken by id to keep paging deterministic across boundaries.
// limit <= 0 means "no limit" (matches ListUsers semantics); callers in
// production pass a sane page size.
func (r *userRepository) ListSystemAdmins(ctx context.Context, offset, limit int) ([]*types.User, int64, error) {
	var users []*types.User
	var total int64

	base := r.db.WithContext(ctx).Model(&types.User{}).Where("is_system_admin = ?", true)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.Order("created_at DESC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// RevokeSystemAdmin revokes system-admin privileges inside a transaction.
// It locks the current admin rows before counting so concurrent revokes
// cannot both observe "two admins" and leave the platform with zero.
//
// Return contract:
//   - (user, nil): revoke actually happened; user.IsSystemAdmin == false
//   - (user, ErrUserNotSystemAdmin): target was already not an admin;
//     no row was written. Caller should treat as idempotent success but
//     MUST distinguish it from a real revoke for audit purposes — the
//     surfaced `user` is the unchanged DB row.
//   - (nil, ErrCannotRevokeSelf | ErrLastSystemAdmin | ErrUserNotFound | …):
//     hard rejection; no row written.
func (r *userRepository) RevokeSystemAdmin(ctx context.Context, userID, actorID string) (*types.User, error) {
	if userID == actorID {
		return nil, ErrCannotRevokeSelf
	}

	var revoked *types.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}
		var user types.User
		if err := locking(tx).
			Where("id = ?", userID).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if !user.IsSystemAdmin {
			revoked = &user
			return ErrUserNotSystemAdmin
		}

		var admins []types.User
		if err := locking(tx).
			Where("is_system_admin = ?", true).
			Find(&admins).Error; err != nil {
			return err
		}
		if len(admins) <= 1 {
			return ErrLastSystemAdmin
		}

		user.IsSystemAdmin = false
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		revoked = &user
		return nil
	})
	// Propagate ErrUserNotSystemAdmin up to the handler alongside the
	// (unchanged) user row. The handler treats it as idempotent success
	// but emits an audit row with changed=false so a probing pattern
	// ("revoke every random user id we know") still leaves a trail.
	if errors.Is(err, ErrUserNotSystemAdmin) {
		return revoked, err
	}
	if err != nil {
		return nil, err
	}
	return revoked, nil
}

// SetSystemRole rewrites the system-level role flags (IsSystemAdmin and
// CanAccessAllTenants) of a user inside a transaction. The actor cannot
// change their own role, and demoting the last system admin is rejected;
// the admin rows are locked before counting for the same reason as in
// RevokeSystemAdmin.
//
// Returns the user as it is after the call, its role before the call and
// whether anything was written (false when the role was already set).
func (r *userRepository) SetSystemRole(
	ctx context.Context, userID, actorID string, role types.SystemRole,
) (*types.User, types.SystemRole, bool, error) {
	if !role.IsValid() {
		return nil, "", false, ErrInvalidSystemRole
	}
	if userID == actorID {
		return nil, "", false, ErrCannotChangeOwnRole
	}

	var (
		result  *types.User
		oldRole types.SystemRole
		changed bool
	)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}
		var user types.User
		if err := locking(tx).
			Where("id = ?", userID).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		oldRole = user.SystemRole()
		result = &user

		isAdmin, allTenants := role.Flags()
		if user.IsSystemAdmin == isAdmin && user.CanAccessAllTenants == allTenants {
			return nil
		}

		if user.IsSystemAdmin && !isAdmin {
			var admins []types.User
			if err := locking(tx).
				Where("is_system_admin = ?", true).
				Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return ErrLastSystemAdmin
			}
		}

		user.IsSystemAdmin = isAdmin
		user.CanAccessAllTenants = allTenants
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return nil, "", false, err
	}
	return result, oldRole, changed, nil
}

// SearchUsers searches users by username or email
func (r *userRepository) SearchUsers(ctx context.Context, query string, limit int) ([]*types.User, error) {
	var users []*types.User
	searchPattern := "%" + query + "%"

	dbQuery := r.db.WithContext(ctx).
		Where("username ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
		Where("is_active = ?", true).
		Order("username ASC")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	} else {
		dbQuery = dbQuery.Limit(20) // default limit
	}

	if err := dbQuery.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// authTokenRepository implements auth token repository interface
type authTokenRepository struct {
	db *gorm.DB
}

// NewAuthTokenRepository creates a new auth token repository
func NewAuthTokenRepository(db *gorm.DB) interfaces.AuthTokenRepository {
	return &authTokenRepository{db: db}
}

// CreateToken creates an auth token
func (r *authTokenRepository) CreateToken(ctx context.Context, token *types.AuthToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetTokenByValue gets a token by its value
func (r *authTokenRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*types.AuthToken, error) {
	var token types.AuthToken
	if err := r.db.WithContext(ctx).Where("token = ?", tokenValue).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// GetTokenByID gets a token by its primary key
func (r *authTokenRepository) GetTokenByID(ctx context.Context, id string) (*types.AuthToken, error) {
	var token types.AuthToken
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// GetTokensByUserID gets all tokens for a user
func (r *authTokenRepository) GetTokensByUserID(ctx context.Context, userID string) ([]*types.AuthToken, error) {
	var tokens []*types.AuthToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// UpdateToken updates a token
func (r *authTokenRepository) UpdateToken(ctx context.Context, token *types.AuthToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

// DeleteToken deletes a token
func (r *authTokenRepository) DeleteToken(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&types.AuthToken{}).Error
}

// DeleteExpiredTokens deletes all expired tokens
func (r *authTokenRepository) DeleteExpiredTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < NOW()").Delete(&types.AuthToken{}).Error
}

// RevokeTokensByUserID revokes all tokens for a user
func (r *authTokenRepository) RevokeTokensByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&types.AuthToken{}).Where("user_id = ?", userID).Update("is_revoked", true).Error
}
