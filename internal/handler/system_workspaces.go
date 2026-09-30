package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/application/repository"
	"github.com/acrbaran/rag/internal/application/service"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	secutils "github.com/acrbaran/rag/internal/utils"
)

// System-admin workspace management (Management → Workspaces tab). Unlike
// the /tenants/:id endpoints, these are not bound to the caller's active
// workspace: RegisterSystemAdminRoutes gates them with SystemAdmin() so an
// operator can inspect and repair any workspace. Membership invariants
// (last owner, duplicate membership) still live in TenantMemberService.

const (
	systemWorkspacesDefaultPageSize = 20
	// Each row fans out into a membership lookup, so keep pages small.
	systemWorkspacesMaxPageSize  = 50
	systemWorkspaceNameMaxLength = 128
)

// SystemWorkspaceOwner identifies one owner of a workspace in the list view.
type SystemWorkspaceOwner struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// SystemWorkspaceListItem is one row of the system-admin workspaces table.
type SystemWorkspaceListItem struct {
	ID          uint64                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	MemberCount int                    `json:"member_count"`
	Owners      []SystemWorkspaceOwner `json:"owners"`
}

// ListSystemWorkspacesResponse is the payload of GET /system/admin/workspaces.
type ListSystemWorkspacesResponse struct {
	Total      int64                      `json:"total"`
	Workspaces []*SystemWorkspaceListItem `json:"workspaces"`
}

// ListSystemWorkspaceMembersResponse is the payload of
// GET /system/admin/workspaces/:id/members.
type ListSystemWorkspaceMembersResponse struct {
	Total   int                          `json:"total"`
	Members []types.TenantMemberResponse `json:"members"`
}

// UpdateSystemWorkspaceRequest renames a workspace. Only the name is
// editable from the system-admin screen.
type UpdateSystemWorkspaceRequest struct {
	Name string `json:"name" binding:"required"`
}

// ListSystemWorkspaces godoc
// @Summary      List all workspaces
// @Description  Paginated list of every workspace with its owners and member count (SystemAdmin only).
// @Description  `q` matches name/description, or the workspace ID when numeric.
// @Tags         System Admin
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size (max 50)" default(20)
// @Param        q         query string false "Search keyword"
// @Success      200  {object}  ListSystemWorkspacesResponse
// @Failure      403  {object}  map[string]interface{}  "Forbidden: not a system admin"
// @Router       /system/admin/workspaces [get]
func (h *SystemHandler) ListSystemWorkspaces(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())

	page := 1
	pageSize := systemWorkspacesDefaultPageSize
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > systemWorkspacesMaxPageSize {
		pageSize = systemWorkspacesMaxPageSize
	}
	search := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(search) > 200 {
		search = string([]rune(search)[:200])
	}
	var searchID uint64
	if id, err := strconv.ParseUint(search, 10, 64); err == nil {
		searchID = id
	}

	tenants, total, err := h.tenantSvc.SearchTenants(ctx, search, searchID, page, pageSize)
	if err != nil {
		logger.Errorf(ctx, "Error listing workspaces: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workspaces"})
		return
	}

	items := make([]*SystemWorkspaceListItem, 0, len(tenants))
	ownerIDsByTenant := make(map[uint64][]string, len(tenants))
	var ownerIDs []string
	for _, t := range tenants {
		item := &SystemWorkspaceListItem{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt,
			Owners:      []SystemWorkspaceOwner{},
		}
		members, err := h.memberSvc.ListByTenant(ctx, t.ID)
		if err != nil {
			// Degrade to an empty row rather than failing the whole page.
			logger.Warnf(ctx, "ListSystemWorkspaces: members lookup failed for tenant %d: %v", t.ID, err)
		}
		item.MemberCount = len(members)
		for _, m := range members {
			if m.Role == types.TenantRoleOwner {
				ownerIDsByTenant[t.ID] = append(ownerIDsByTenant[t.ID], m.UserID)
				ownerIDs = append(ownerIDs, m.UserID)
			}
		}
		items = append(items, item)
	}

	usersByID := h.lookupUsers(ctx, ownerIDs)
	for _, item := range items {
		for _, uid := range ownerIDsByTenant[item.ID] {
			owner := SystemWorkspaceOwner{UserID: uid}
			if u := usersByID[uid]; u != nil {
				owner.Email = u.Email
				owner.Username = u.Username
			}
			item.Owners = append(item.Owners, owner)
		}
	}

	c.JSON(http.StatusOK, ListSystemWorkspacesResponse{Total: total, Workspaces: items})
}

// UpdateSystemWorkspace godoc
// @Summary      Rename a workspace
// @Description  Change any workspace's name (SystemAdmin only).
// @Tags         System Admin
// @Accept       json
// @Produce      json
// @Param        id      path int                          true "Workspace ID"
// @Param        request body UpdateSystemWorkspaceRequest true "New name"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}  "Blank or too long name"
// @Failure      404  {object}  map[string]interface{}  "Workspace not found"
// @Router       /system/admin/workspaces/{id} [put]
func (h *SystemHandler) UpdateSystemWorkspace(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	var req UpdateSystemWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace name cannot be blank"})
		return
	}
	if utf8.RuneCountInString(name) > systemWorkspaceNameMaxLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace name is too long"})
		return
	}

	oldName := tenant.Name
	if name != oldName {
		tenant.Name = name
		if _, err := h.tenantSvc.UpdateTenant(ctx, tenant); err != nil {
			logger.Errorf(ctx, "Error renaming workspace %d: %v", tenant.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workspace"})
			return
		}
		logger.Infof(ctx, "Workspace %d renamed by system administrator to %s",
			tenant.ID, secutils.SanitizeForLog(name))
		h.emitWorkspaceAudit(ctx, types.AuditActionSystemWorkspaceUpdated, tenant, map[string]any{
			"old_name": oldName,
			"new_name": name,
		})
	}

	c.JSON(http.StatusOK, gin.H{"id": tenant.ID, "name": tenant.Name})
}

// DeleteSystemWorkspace godoc
// @Summary      Delete a workspace
// @Description  Delete any workspace together with all of its memberships (SystemAdmin only).
// @Tags         System Admin
// @Produce      json
// @Param        id path int true "Workspace ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}  "Workspace not found"
// @Router       /system/admin/workspaces/{id} [delete]
func (h *SystemHandler) DeleteSystemWorkspace(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	// Snapshot members before the memberships are purged: the audit row
	// records how many people lost access, and their user rows may still
	// point at this workspace afterwards.
	members, err := h.memberSvc.ListByTenant(ctx, tenant.ID)
	if err != nil {
		logger.Warnf(ctx, "DeleteSystemWorkspace: members snapshot failed for tenant %d: %v", tenant.ID, err)
	}

	if err := h.tenantSvc.DeleteTenant(ctx, tenant.ID); err != nil {
		logger.Errorf(ctx, "Error deleting workspace %d: %v", tenant.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete workspace"})
		return
	}

	for _, m := range members {
		h.clearStaleWorkspacePointers(ctx, m.UserID, tenant.ID)
	}

	logger.Infof(ctx, "Workspace %d deleted by system administrator", tenant.ID)
	h.emitWorkspaceAudit(ctx, types.AuditActionSystemWorkspaceDeleted, tenant, map[string]any{
		"workspace_name":  tenant.Name,
		"members_removed": len(members),
	})
	c.JSON(http.StatusOK, gin.H{"message": "Workspace deleted successfully"})
}

// ListSystemWorkspaceMembers godoc
// @Summary      List a workspace's members
// @Description  Every member of the workspace with their role (SystemAdmin only).
// @Tags         System Admin
// @Produce      json
// @Param        id path int true "Workspace ID"
// @Success      200  {object}  ListSystemWorkspaceMembersResponse
// @Failure      404  {object}  map[string]interface{}  "Workspace not found"
// @Router       /system/admin/workspaces/{id}/members [get]
func (h *SystemHandler) ListSystemWorkspaceMembers(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	members, err := h.memberSvc.ListByTenant(ctx, tenant.ID)
	if err != nil {
		logger.Errorf(ctx, "Error listing members of workspace %d: %v", tenant.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list members"})
		return
	}

	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}
	usersByID := h.lookupUsers(ctx, ids)

	resp := make([]types.TenantMemberResponse, 0, len(members))
	for _, m := range members {
		row := types.TenantMemberResponse{
			UserID:    m.UserID,
			Role:      m.Role,
			Status:    m.Status,
			InvitedBy: m.InvitedBy,
			JoinedAt:  m.JoinedAt,
		}
		if u := usersByID[m.UserID]; u != nil {
			row.Email = u.Email
			row.Username = u.Username
			row.Avatar = u.Avatar
		}
		resp = append(resp, row)
	}

	c.JSON(http.StatusOK, ListSystemWorkspaceMembersResponse{Total: len(resp), Members: resp})
}

// AddSystemWorkspaceMember godoc
// @Summary      Add a member to a workspace
// @Description  Add a registered user to any workspace by email with the given role (SystemAdmin only).
// @Tags         System Admin
// @Accept       json
// @Produce      json
// @Param        id      path int              true "Workspace ID"
// @Param        request body addMemberRequest true "Email and role"
// @Success      201  {object}  types.TenantMemberResponse
// @Failure      404  {object}  map[string]interface{}  "Workspace or user not found"
// @Failure      409  {object}  map[string]interface{}  "Already a member"
// @Router       /system/admin/workspaces/{id}/members [post]
func (h *SystemHandler) AddSystemWorkspaceMember(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if !req.Role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace role"})
		return
	}

	user, err := h.userSvc.GetUserByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No registered user with this email"})
			return
		}
		logger.Errorf(ctx, "GetUserByEmail failed: email=%s err=%v", secutils.SanitizeForLog(req.Email), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to look up user"})
		return
	}

	var invitedBy *string
	if caller, _ := types.UserIDFromContext(ctx); caller != "" && !types.IsSyntheticUserID(caller) {
		invitedBy = &caller
	}
	member, err := h.memberSvc.AddMember(ctx, user.ID, tenant.ID, req.Role, invitedBy)
	if err != nil {
		h.writeSystemMemberError(c, ctx, err, "Failed to add member")
		return
	}

	c.JSON(http.StatusCreated, types.TenantMemberResponse{
		UserID:    member.UserID,
		Email:     user.Email,
		Username:  user.Username,
		Avatar:    user.Avatar,
		Role:      member.Role,
		Status:    member.Status,
		InvitedBy: member.InvitedBy,
		JoinedAt:  member.JoinedAt,
	})
}

// CreateSystemWorkspaceInviteLink godoc
// @Summary      Create a share-link invitation for a workspace
// @Description  Issue a multi-use share-link invitation for any workspace with the given role (SystemAdmin only).
// @Description  Anyone opening the link can register (or sign in) and join the workspace until it expires or is revoked.
// @Tags         System Admin
// @Accept       json
// @Produce      json
// @Param        id      path int                     true "Workspace ID"
// @Param        request body createInviteLinkRequest true "Role for link holders"
// @Success      201  {object}  types.TenantInvitationResponse
// @Failure      404  {object}  map[string]interface{}  "Workspace not found"
// @Router       /system/admin/workspaces/{id}/invite-links [post]
func (h *SystemHandler) CreateSystemWorkspaceInviteLink(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	if h.invitationSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Invitations are not available"})
		return
	}
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	var req createInviteLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if !req.Role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace role"})
		return
	}

	var invitedBy *string
	if caller, _ := types.UserIDFromContext(ctx); caller != "" && !types.IsSyntheticUserID(caller) {
		invitedBy = &caller
	}
	inv, token, err := h.invitationSvc.CreateShareLink(ctx, tenant.ID, req.Role, invitedBy, req.Message)
	if err != nil {
		logger.Errorf(ctx, "CreateShareLink failed: tenant=%d err=%v", tenant.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invite link"})
		return
	}

	usersByID := map[string]*types.User{}
	if invitedBy != nil {
		usersByID = h.lookupUsers(ctx, []string{*invitedBy})
	}
	resp := projectInvitation(inv, usersByID, map[uint64]*types.Tenant{tenant.ID: tenant})
	resp.InviteURL = buildInviteRegisterURL(h.cfg, token)
	c.JSON(http.StatusCreated, resp)
}

// ListSystemWorkspaceInvitations godoc
// @Summary      List a workspace's pending invitations
// @Description  Pending per-user invitations and share links of any workspace. Share-link rows carry invite_url so the admin can re-copy them.
// @Tags         System Admin
// @Produce      json
// @Param        id        path  int  true  "Workspace ID"
// @Param        page      query int  false "Page number" default(1)
// @Param        page_size query int  false "Page size (max 100)" default(20)
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}  "Workspace not found"
// @Router       /system/admin/workspaces/{id}/invitations [get]
func (h *SystemHandler) ListSystemWorkspaceInvitations(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	if h.invitationSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Invitations are not available"})
		return
	}
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}

	page := 1
	pageSize := defaultListPageSize
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n > 0 {
		page = n
	}
	if n, err := strconv.Atoi(c.Query("page_size")); err == nil && n > 0 {
		pageSize = n
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}

	rows, total, err := h.invitationSvc.ListTenantInvitationsPage(ctx, tenant.ID, false, page, pageSize)
	if err != nil {
		logger.Errorf(ctx, "ListTenantInvitationsPage failed: tenant=%d err=%v", tenant.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list invitations"})
		return
	}

	idSet := make(map[string]struct{}, len(rows)*2)
	for _, inv := range rows {
		if inv.InviteeUserID != "" {
			idSet[inv.InviteeUserID] = struct{}{}
		}
		if inv.InvitedBy != nil {
			idSet[*inv.InvitedBy] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	usersByID := h.lookupUsers(ctx, ids)

	resp := make([]types.TenantInvitationResponse, 0, len(rows))
	for _, inv := range rows {
		item := projectInvitation(inv, usersByID, nil)
		if inv.Status == types.TenantInvitationStatusPending && inv.Token != "" {
			item.InviteURL = buildInviteRegisterURL(h.cfg, inv.Token)
		}
		resp = append(resp, item)
	}
	c.JSON(http.StatusOK, gin.H{
		"invitations": resp,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// RevokeSystemWorkspaceInvitation godoc
// @Summary      Revoke a workspace invitation
// @Description  Revokes a pending invitation or share link of any workspace.
// @Tags         System Admin
// @Produce      json
// @Param        id      path int true "Workspace ID"
// @Param        inv_id  path int true "Invitation ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}  "Workspace or invitation not found"
// @Failure      409  {object}  map[string]interface{}  "Invitation is no longer pending"
// @Router       /system/admin/workspaces/{id}/invitations/{inv_id} [delete]
func (h *SystemHandler) RevokeSystemWorkspaceInvitation(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	if h.invitationSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Invitations are not available"})
		return
	}
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}
	invID, err := strconv.ParseUint(c.Param("inv_id"), 10, 64)
	if err != nil || invID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	// The invitation row carries its own tenant_id; a mismatch renders as
	// missing so the URL :id can't be used to reach another workspace's rows.
	inv, err := h.invitationSvc.GetByID(ctx, invID)
	if err != nil {
		logger.Errorf(ctx, "GetByID invitation failed: id=%d err=%v", invID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke invitation"})
		return
	}
	if inv == nil || inv.TenantID != tenant.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	if err := h.invitationSvc.Revoke(ctx, invID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		case errors.Is(err, service.ErrInvitationNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "Invitation is no longer pending"})
		default:
			logger.Errorf(ctx, "RevokeInvitation failed: id=%d err=%v", invID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke invitation"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateSystemWorkspaceMemberRole godoc
// @Summary      Change a member's workspace role
// @Description  Change the role of a member inside any workspace (SystemAdmin only). The last owner cannot be demoted.
// @Tags         System Admin
// @Accept       json
// @Produce      json
// @Param        id      path int                     true "Workspace ID"
// @Param        user_id path string                  true "User ID"
// @Param        request body updateMemberRoleRequest true "New role"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}  "Workspace or membership not found"
// @Failure      409  {object}  map[string]interface{}  "Last owner"
// @Router       /system/admin/workspaces/{id}/members/{user_id} [put]
func (h *SystemHandler) UpdateSystemWorkspaceMemberRole(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	var req updateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if !req.Role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace role"})
		return
	}

	if err := h.memberSvc.UpdateRole(ctx, userID, tenant.ID, req.Role); err != nil {
		h.writeSystemMemberError(c, ctx, err, "Failed to update member role")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member role updated"})
}

// RemoveSystemWorkspaceMember godoc
// @Summary      Remove a member from a workspace
// @Description  Remove a member from any workspace (SystemAdmin only). The removed user's sessions are
// @Description  revoked, so an administrator cannot remove themselves here. The last owner cannot be removed.
// @Tags         System Admin
// @Produce      json
// @Param        id      path int    true "Workspace ID"
// @Param        user_id path string true "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}  "Self removal"
// @Failure      404  {object}  map[string]interface{}  "Workspace or membership not found"
// @Failure      409  {object}  map[string]interface{}  "Last owner"
// @Router       /system/admin/workspaces/{id}/members/{user_id} [delete]
func (h *SystemHandler) RemoveSystemWorkspaceMember(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	tenant, ok := h.loadSystemWorkspace(c, ctx)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	// RemoveMember revokes every token of the removed user; doing that to
	// the caller would sign the administrator out mid-operation.
	if caller, _ := types.UserIDFromContext(ctx); caller == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove yourself from a workspace here"})
		return
	}

	if err := h.memberSvc.RemoveMember(ctx, userID, tenant.ID); err != nil {
		h.writeSystemMemberError(c, ctx, err, "Failed to remove member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// loadSystemWorkspace resolves :id to an existing workspace. On false the
// response has already been written and the caller must return.
func (h *SystemHandler) loadSystemWorkspace(c *gin.Context, ctx context.Context) (*types.Tenant, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return nil, false
	}
	tenant, err := h.tenantSvc.GetTenantByID(ctx, id)
	if err != nil || tenant == nil {
		if err == nil || errors.Is(err, repository.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
			return nil, false
		}
		logger.Errorf(ctx, "Error loading workspace %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load workspace"})
		return nil, false
	}
	return tenant, true
}

// clearStaleWorkspacePointers drops users.tenant_id / last-active
// preferences that still reference a deleted workspace, so the next login
// resolves a live membership instead. Tokens are left alone: a JWT bound to
// the deleted workspace is already rejected by the auth middleware, and
// revoking would also sign out the administrator performing the delete.
func (h *SystemHandler) clearStaleWorkspacePointers(ctx context.Context, userID string, tenantID uint64) {
	user, err := h.userSvc.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return
	}
	changed := false
	if user.TenantID == tenantID {
		user.TenantID = 0
		changed = true
	}
	if user.Preferences.LastActiveTenantID != nil && *user.Preferences.LastActiveTenantID == tenantID {
		user.Preferences.LastActiveTenantID = nil
		changed = true
	}
	if !changed {
		return
	}
	user.UpdatedAt = time.Now()
	if err := h.userSvc.UpdateUser(ctx, user); err != nil {
		logger.Warnf(ctx, "DeleteSystemWorkspace: failed to clear tenant pointers for user %s: %v", userID, err)
	}
}

// lookupUsers batch-loads users by id. Failure is best-effort: rows render
// without email/username instead of failing the whole response.
func (h *SystemHandler) lookupUsers(ctx context.Context, ids []string) map[string]*types.User {
	if len(ids) == 0 {
		return map[string]*types.User{}
	}
	users, err := h.userSvc.GetUsersByIDs(ctx, ids)
	if err != nil {
		logger.Warnf(ctx, "Batch user lookup failed: %v", err)
		return map[string]*types.User{}
	}
	return users
}

// writeSystemMemberError maps TenantMemberService sentinels to HTTP responses.
func (h *SystemHandler) writeSystemMemberError(c *gin.Context, ctx context.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrMembershipNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Membership not found"})
	case errors.Is(err, service.ErrMembershipAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this workspace"})
	case errors.Is(err, service.ErrLastOwner):
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot demote or remove the last owner of the workspace"})
	case errors.Is(err, service.ErrInvalidTenantRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace role"})
	default:
		logger.Errorf(ctx, "%s: %v", fallback, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

// emitWorkspaceAudit writes one system-scope audit row for a workspace
// lifecycle event. Best-effort, same semantics as emitAdminAudit.
func (h *SystemHandler) emitWorkspaceAudit(
	ctx context.Context,
	action types.AuditAction,
	tenant *types.Tenant,
	details map[string]any,
) {
	if h.auditSvc == nil || tenant == nil {
		return
	}
	actorID, _ := types.UserIDFromContext(ctx)
	var detailsJSON types.JSON
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			detailsJSON = types.JSON(b)
		}
	}
	_ = h.auditSvc.Log(ctx, &types.AuditLog{
		TenantID:    0,
		ActorUserID: actorID,
		ActorRole:   systemAuditActorRole(ctx),
		Action:      action,
		TargetType:  "tenant",
		TargetID:    strconv.FormatUint(tenant.ID, 10),
		Outcome:     types.AuditOutcomeSuccess,
		Details:     detailsJSON,
	})
}
