package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
	"github.com/johnnycube/openbeehive-app/server/internal/storage"
)

// TenantAPI exposes multi-tenant onboarding: who am I, which tenants can I use,
// switch the active tenant, create a tenant, invite users, accept an invite.
type TenantAPI struct {
	sessions *SessionManager
	users    storage.UserRepo
	orgs     storage.OrgRepo
	members  storage.MemberRepo
	invites  storage.InviteRepo
	prov     *Provisioner
	cfg      *config.Config
}

func NewTenantAPI(s *SessionManager, u storage.UserRepo, o storage.OrgRepo, m storage.MemberRepo, i storage.InviteRepo, prov *Provisioner, cfg *config.Config) *TenantAPI {
	return &TenantAPI{sessions: s, users: u, orgs: o, members: m, invites: i, prov: prov, cfg: cfg}
}

func (t *TenantAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/instance", t.instance)
	mux.HandleFunc("/auth/me", t.me)
	mux.HandleFunc("/auth/switch", t.switchTenant)
	mux.HandleFunc("/tenants/create", t.create)
	mux.HandleFunc("/tenants/invite", t.invite)
	mux.HandleFunc("/auth/accept-invite", t.accept)
}

// instance advertises onboarding state + every enabled sign-in method, so the
// login screen can render exactly what's available.
func (t *TenantAPI) instance(w http.ResponseWriter, r *http.Request) {
	n, err := realUserCount(r.Context(), t.users, t.cfg)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	providers := t.cfg.OIDC.Enabled
	if providers == nil {
		providers = []string{}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"needs_setup":        n == 0, // first run: the next sign-up is the admin
		"password_auth":      t.cfg.Auth.PasswordEnabled,
		"registration":       t.cfg.Auth.RegistrationOpen,
		"email_verification": t.cfg.Auth.EmailVerification,
		"oidc_providers":     providers,
		"webauthn":           t.cfg.Auth.WebAuthn.Enabled,
		"demo":               t.cfg.Demo.Enabled,
	})
}

func (t *TenantAPI) me(w http.ResponseWriter, r *http.Request) {
	id, ok := t.sessions.IdentityFromRequest(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]any{"error": "not signed in"})
		return
	}
	tenants, _ := t.members.ListByUser(r.Context(), id.UserID)
	list := make([]map[string]any, 0, len(tenants))
	for _, m := range tenants {
		list = append(list, map[string]any{"id": m.OrgID, "name": m.Name, "role": m.Role})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"user_id":        id.UserID,
		"email":          id.Email,
		"instance_admin": id.Role == "admin",
		"active_org":     id.OrgID,
		"is_demo":        t.cfg.Demo.Enabled && strings.EqualFold(id.Email, t.cfg.Demo.Email),
		"tenants":        list,
	})
}

type switchReq struct {
	OrgID string `json:"org_id"`
}

func (t *TenantAPI) switchTenant(w http.ResponseWriter, r *http.Request) {
	id, ok := t.sessions.IdentityFromRequest(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]any{"error": "not signed in"})
		return
	}
	var req switchReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if _, err := t.members.Get(r.Context(), req.OrgID, id.UserID); err != nil {
		respondJSON(w, http.StatusForbidden, map[string]any{"error": "not a member of that tenant"})
		return
	}
	tok := t.reissue(w, id, req.OrgID)
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "active_org": req.OrgID, "token": tok})
}

type createReq struct {
	Name string `json:"name"`
}

func (t *TenantAPI) create(w http.ResponseWriter, r *http.Request) {
	id, ok := t.sessions.IdentityFromRequest(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]any{"error": "not signed in"})
		return
	}
	if id.Role == "demo" {
		respondJSON(w, http.StatusForbidden, map[string]any{"error": ErrDemoReadOnly.Error()})
		return
	}
	var req createReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "a tenant name is required"})
		return
	}
	org := &storage.Organization{ID: uuid.NewString(), Name: name, Plan: "hobby", CreatedAt: time.Now().UTC()}
	if err := t.orgs.Create(r.Context(), org); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	// Creator becomes the tenant admin (owner) and switches into the new tenant.
	if err := t.members.Add(r.Context(), &storage.Membership{OrgID: org.ID, UserID: id.UserID, Role: "owner"}); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	tok := t.reissue(w, id, org.ID)
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "org_id": org.ID, "name": org.Name, "active_org": org.ID, "token": tok})
}

type inviteReq struct {
	OrgID string `json:"org_id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (t *TenantAPI) invite(w http.ResponseWriter, r *http.Request) {
	id, ok := t.sessions.IdentityFromRequest(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]any{"error": "not signed in"})
		return
	}
	if id.Role == "demo" {
		respondJSON(w, http.StatusForbidden, map[string]any{"error": ErrDemoReadOnly.Error()})
		return
	}
	var req inviteReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.OrgID == "" {
		req.OrgID = id.OrgID
	}
	// Only a tenant admin (owner) may invite.
	m, err := t.members.Get(r.Context(), req.OrgID, id.UserID)
	if err != nil || m.Role != "owner" {
		respondJSON(w, http.StatusForbidden, map[string]any{"error": "only the tenant admin can invite"})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(email, "@") {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "a valid email is required"})
		return
	}
	role := req.Role
	if role != "owner" {
		role = "member"
	}
	inv := &storage.Invite{ID: uuid.NewString(), OrgID: req.OrgID, Email: email, Role: role, Token: randomToken(), CreatedAt: time.Now().UTC()}
	if err := t.invites.Create(r.Context(), inv); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	link := fmt.Sprintf("%s/login?invite=%s", strings.TrimRight(t.cfg.PublicBaseURL, "/"), inv.Token)
	log.Printf("tenant: invite for %s to tenant %s: %s", email, req.OrgID, link)
	// (Email delivery reuses the SMTP config when set; logged otherwise.)
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "token": inv.Token, "link": link})
}

type acceptReq struct {
	Token string `json:"token"`
}

func (t *TenantAPI) accept(w http.ResponseWriter, r *http.Request) {
	id, ok := t.sessions.IdentityFromRequest(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]any{"error": "not signed in"})
		return
	}
	if id.Role == "demo" {
		respondJSON(w, http.StatusForbidden, map[string]any{"error": ErrDemoReadOnly.Error()})
		return
	}
	var req acceptReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	inv, err := t.invites.GetByToken(r.Context(), req.Token)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid or expired invite"})
		return
	}
	if _, err := t.members.Get(r.Context(), inv.OrgID, id.UserID); err != nil {
		// not yet a member -> add
		if err := t.members.Add(r.Context(), &storage.Membership{OrgID: inv.OrgID, UserID: id.UserID, Role: inv.Role}); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	_ = t.invites.Delete(r.Context(), inv.ID)
	tok := t.reissue(w, id, inv.OrgID) // switch into the joined tenant
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "org_id": inv.OrgID, "active_org": inv.OrgID, "token": tok})
}

// reissue mints a fresh session for the same user with a new active tenant,
// sets the cookie and returns the token (for Bearer clients).
func (t *TenantAPI) reissue(w http.ResponseWriter, id Identity, orgID string) string {
	tok, err := t.sessions.Issue(Identity{UserID: id.UserID, OrgID: orgID, Email: id.Email, Role: id.Role})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return ""
	}
	t.sessions.SetCookie(w, tok)
	return tok
}
