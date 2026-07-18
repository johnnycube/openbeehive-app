package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
	"github.com/johnnycube/openbeehive-app/server/internal/storage"
	sqlstore "github.com/johnnycube/openbeehive-app/server/internal/storage/sql"
)

func newAuthFixture(t *testing.T, cfg *config.Config) (*httptest.Server, *sqlstore.Store, *SessionManager) {
	t.Helper()
	cfg.DB.Driver = config.DriverSQLite
	cfg.DB.DSN = "file:" + filepath.Join(t.TempDir(), "auth.db")
	store, err := sqlstore.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sessions := NewSessionManager("test-secret", time.Hour)
	prov := NewProvisioner(store.Users(), store.Orgs(), store.Members())
	mux := http.NewServeMux()
	NewPasswordAuth(store.Users(), store.Invites(), sessions, cfg, prov).Routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, store, sessions
}

func postJSON(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestSignupInviteOnly(t *testing.T) {
	cfg := &config.Config{Profile: "selfhost"}
	cfg.Auth.PasswordEnabled = true
	cfg.Auth.RegistrationOpen = false
	srv, store, _ := newAuthFixture(t, cfg)

	// First-run setup is always allowed and creates the admin.
	code, j := postJSON(t, srv.URL+"/auth/signup",
		map[string]string{"email": "admin@test.dev", "password": "password123", "name": "Admin"})
	if code != http.StatusOK || j["admin"] != true {
		t.Fatalf("first signup: code=%d resp=%v", code, j)
	}

	// Open registration is closed for everyone after that.
	code, j = postJSON(t, srv.URL+"/auth/signup",
		map[string]string{"email": "stranger@test.dev", "password": "password123"})
	if code != http.StatusForbidden || j["status"] != "invite_only" {
		t.Fatalf("stranger signup: code=%d resp=%v", code, j)
	}

	// A valid invite for the matching email passes.
	inv := &storage.Invite{ID: "inv1", OrgID: "org1", Email: "friend@test.dev",
		Role: "member", Token: "tok-friend", CreatedAt: time.Now().UTC()}
	if err := store.Invites().Create(context.Background(), inv); err != nil {
		t.Fatalf("invite create: %v", err)
	}
	code, j = postJSON(t, srv.URL+"/auth/signup", map[string]string{
		"email": "other@test.dev", "password": "password123", "invite": "tok-friend"})
	if code != http.StatusForbidden {
		t.Fatalf("invite with wrong email must be rejected: code=%d resp=%v", code, j)
	}
	code, j = postJSON(t, srv.URL+"/auth/signup", map[string]string{
		"email": "friend@test.dev", "password": "password123", "invite": "tok-friend"})
	if code != http.StatusOK {
		t.Fatalf("invited signup: code=%d resp=%v", code, j)
	}

	// Existing accounts keep signing in.
	code, _ = postJSON(t, srv.URL+"/auth/signin",
		map[string]string{"email": "admin@test.dev", "password": "password123"})
	if code != http.StatusOK {
		t.Fatalf("admin signin: code=%d", code)
	}
}

func TestDemoInstanceDisablesPublicSignup(t *testing.T) {
	cfg := &config.Config{Profile: "selfhost"}
	cfg.Auth.PasswordEnabled = true
	cfg.Auth.RegistrationOpen = false
	cfg.Demo = config.DemoConfig{Enabled: true, Email: "demo@test.dev", Password: "demo"}
	srv, store, _ := newAuthFixture(t, cfg)

	// The demo account exists before anyone signs up (installed at boot).
	demo := &storage.User{ID: "demo-user", Email: "demo@test.dev", Name: "Demo",
		Role: "user", EmailVerified: true, CreatedAt: time.Now().UTC()}
	if err := store.Users().Create(context.Background(), demo); err != nil {
		t.Fatalf("demo user: %v", err)
	}

	// Public sign-up is disabled on a demo instance. Without this, the seeded
	// demo user is excluded from realUserCount, so the instance looks like it
	// "needs setup" and the first visitor to POST /auth/signup would create the
	// instance-admin account (first-run setup bypasses invite-only). The demo
	// login is the only door; no account may be created via the public URL.
	code, j := postJSON(t, srv.URL+"/auth/signup",
		map[string]string{"email": "admin@test.dev", "password": "password123"})
	if code != http.StatusForbidden {
		t.Fatalf("signup on demo instance must be forbidden: code=%d resp=%v", code, j)
	}

	// And no account was created as a side effect.
	if _, err := store.Users().GetByEmail(context.Background(), "admin@test.dev"); err == nil {
		t.Fatalf("signup on demo instance must not create an account")
	}
}

func TestDemoSessionIsMarkedReadOnly(t *testing.T) {
	cfg := &config.Config{Profile: "selfhost"}
	cfg.Auth.PasswordEnabled = true
	cfg.Auth.RegistrationOpen = true
	cfg.Demo = config.DemoConfig{Enabled: true, Email: "demo@test.dev", Password: "demo"}
	srv, store, sessions := newAuthFixture(t, cfg)

	hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	u := &storage.User{ID: "demo-user", Email: "demo@test.dev", Name: "Demo",
		Role: "user", PasswordHash: string(hash), EmailVerified: true, CreatedAt: time.Now().UTC()}
	if err := store.Users().Create(context.Background(), u); err != nil {
		t.Fatalf("user create: %v", err)
	}

	for path, body := range map[string]map[string]string{
		"/auth/demo-login": {},
		"/auth/signin":     {"email": "demo@test.dev", "password": "demo1234"},
	} {
		code, j := postJSON(t, srv.URL+path, body)
		if code != http.StatusOK {
			t.Fatalf("%s: code=%d resp=%v", path, code, j)
		}
		tok, _ := j["token"].(string)
		id, err := sessions.Verify(context.Background(), tok)
		if err != nil {
			t.Fatalf("%s: verify: %v", path, err)
		}
		if id.Role != "demo" {
			t.Fatalf("%s: session role = %q, want demo", path, id.Role)
		}
	}
}

func TestReadOnlyMethodAllowList(t *testing.T) {
	reads := []string{
		"/openbeehive.v1.SyncService/Pull",
		"/openbeehive.v1.SyncService/Subscribe",
		"/openbeehive.v1.ApiaryService/ListApiaries",
		"/openbeehive.v1.ApiaryService/GetApiary",
	}
	writes := []string{
		"/openbeehive.v1.SyncService/Push",
		"/openbeehive.v1.ApiaryService/CreateApiary",
		"/openbeehive.v1.ApiaryService/UpdateApiary",
		"/openbeehive.v1.ApiaryService/DeleteApiary",
	}
	for _, p := range reads {
		if !readOnlyMethod(p) {
			t.Errorf("%s should be read-only", p)
		}
	}
	for _, p := range writes {
		if readOnlyMethod(p) {
			t.Errorf("%s must count as a write", p)
		}
	}
}
