package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"github.com/johnnycube/openbeehive-app/server/internal/auth"
	"github.com/johnnycube/openbeehive-app/server/internal/config"
	wv1 "github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1"
	sqlstore "github.com/johnnycube/openbeehive-app/server/internal/storage/sql"
	wsync "github.com/johnnycube/openbeehive-app/server/internal/sync"
)

// newSyncFixture opens a fresh SQLite store with two tenants, one apiary each,
// and one hive owned by tenant A.
func newSyncFixture(t *testing.T) *SyncService {
	t.Helper()
	cfg := &config.Config{Profile: "selfhost"}
	cfg.DB.Driver = config.DriverSQLite
	cfg.DB.DSN = "file:" + filepath.Join(t.TempDir(), "sync.db")
	store, err := sqlstore.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db := store.DB()
	seed := []string{
		`INSERT INTO apiary (id, organization_id, name, created_at, updated_at)
		 VALUES ('apiary-a', 'org-a', 'A', '2026-01-01', '2026-01-01')`,
		`INSERT INTO apiary (id, organization_id, name, created_at, updated_at)
		 VALUES ('apiary-b', 'org-b', 'B', '2026-01-01', '2026-01-01')`,
		`INSERT INTO hive (id, organization_id, apiary_id, name, created_at, updated_at)
		 VALUES ('hive-a', 'org-a', 'apiary-a', 'H1', '2026-01-01', '2026-01-01')`,
	}
	for _, q := range seed {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return NewSyncService(db, wsync.NewHLC("test"))
}

func identityCtx(user, org string) context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{UserID: user, OrgID: org})
}

func push(svc *SyncService, ctx context.Context, changes ...*wv1.Change) error {
	_, err := svc.Push(ctx, connect.NewRequest(&wv1.PushRequest{Changes: changes}))
	return err
}

func change(entity, id, scope, payload string) *wv1.Change {
	return &wv1.Change{
		Entity: entity, EntityId: id, ScopeId: scope,
		Op: wv1.ChangeOp_CHANGE_OP_UPSERT, PayloadJson: payload,
		Hlc: wsync.NewHLC("dev").Now(), AuthorId: "dev",
	}
}

func wantPermissionDenied(t *testing.T, err error) {
	t.Helper()
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodePermissionDenied {
		t.Fatalf("want PermissionDenied, got %v", err)
	}
}

func TestPushRejectsForeignScope(t *testing.T) {
	svc := newSyncFixture(t)
	// User of tenant B labels a change with tenant A's apiary scope.
	err := push(svc, identityCtx("user-b", "org-b"),
		change("hive", "hive-x", "apiary-a", `{"name":"intruder"}`))
	wantPermissionDenied(t, err)
}

func TestPushRejectsForeignRow(t *testing.T) {
	svc := newSyncFixture(t)
	// Scope label is user B's own apiary, but the row id belongs to tenant A.
	err := push(svc, identityCtx("user-b", "org-b"),
		change("hive", "hive-a", "apiary-b", `{"name":"hijacked"}`))
	wantPermissionDenied(t, err)

	var name string
	if err := svc.db.Get(&name, `SELECT name FROM hive WHERE id = 'hive-a'`); err != nil || name != "H1" {
		t.Fatalf("victim row was modified: name=%q err=%v", name, err)
	}
}

func TestPushRejectsForeignOrgPayload(t *testing.T) {
	svc := newSyncFixture(t)
	// Insert claiming to belong to tenant A while authenticated as tenant B.
	err := push(svc, identityCtx("user-b", "org-b"),
		change("hive", "hive-y", "apiary-b", `{"organization_id":"org-a","name":"smuggled"}`))
	wantPermissionDenied(t, err)
}

func TestPushStampsCallerTenant(t *testing.T) {
	svc := newSyncFixture(t)
	// A legitimate insert (no organization_id in the payload) lands in the
	// caller's tenant.
	if err := push(svc, identityCtx("user-a", "org-a"),
		change("hive", "hive-new", "apiary-a", `{"apiary_id":"apiary-a","name":"New","created_at":"2026-07-11","updated_at":"2026-07-11"}`)); err != nil {
		t.Fatalf("push: %v", err)
	}
	var org string
	if err := svc.db.Get(&org, `SELECT organization_id FROM hive WHERE id = 'hive-new'`); err != nil {
		t.Fatalf("row missing: %v", err)
	}
	if org != "org-a" {
		t.Fatalf("organization_id = %q, want org-a", org)
	}
}

func TestPushAllowsNewApiaryScope(t *testing.T) {
	svc := newSyncFixture(t)
	// A brand-new apiary opens its own scope (scope id = apiary id).
	if err := push(svc, identityCtx("user-a", "org-a"),
		change("apiary", "apiary-new", "apiary-new", `{"name":"Orchard","created_at":"2026-07-11","updated_at":"2026-07-11"}`)); err != nil {
		t.Fatalf("push: %v", err)
	}
	var org string
	if err := svc.db.Get(&org, `SELECT organization_id FROM apiary WHERE id = 'apiary-new'`); err != nil || org != "org-a" {
		t.Fatalf("new apiary org = %q err=%v, want org-a", org, err)
	}
}

func TestPullFiltersByScope(t *testing.T) {
	svc := newSyncFixture(t)
	if err := push(svc, identityCtx("user-a", "org-a"),
		change("hive", "hive-a", "apiary-a", `{"name":"A renamed"}`)); err != nil {
		t.Fatalf("push a: %v", err)
	}
	if err := push(svc, identityCtx("user-b", "org-b"),
		change("hive", "hive-b", "apiary-b", `{"apiary_id":"apiary-b","name":"B hive","created_at":"2026-07-11","updated_at":"2026-07-11"}`)); err != nil {
		t.Fatalf("push b: %v", err)
	}
	resp, err := svc.Pull(identityCtx("user-b", "org-b"),
		connect.NewRequest(&wv1.PullRequest{}))
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	for _, ch := range resp.Msg.Changes {
		if ch.ScopeId != "apiary-b" && ch.ScopeId != "user:user-b" {
			t.Fatalf("pull leaked foreign scope %q", ch.ScopeId)
		}
	}
	if len(resp.Msg.Changes) != 1 {
		t.Fatalf("want exactly the own change, got %d", len(resp.Msg.Changes))
	}
}
