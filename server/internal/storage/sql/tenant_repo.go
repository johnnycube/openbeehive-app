package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/johnnycube/openbeehive-app/server/internal/storage"
)

// --- Organizations (tenants) ---

type orgRepo struct{ s *Store }

func (r *orgRepo) Create(ctx context.Context, o *storage.Organization) error {
	return r.s.exec(ctx, `INSERT INTO organization (id, name, plan, created_at) VALUES (?, ?, ?, ?)`,
		o.ID, o.Name, o.Plan, o.CreatedAt)
}

func (r *orgRepo) Get(ctx context.Context, id string) (*storage.Organization, error) {
	var o storage.Organization
	err := r.s.db.GetContext(ctx, &o, r.s.db.Rebind(
		`SELECT id, name, plan, created_at FROM organization WHERE id = ?`), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	return &o, err
}

// --- Memberships ---

type memberRepo struct{ s *Store }

func (r *memberRepo) Add(ctx context.Context, m *storage.Membership) error {
	return r.s.exec(ctx, `INSERT INTO member (organization_id, benutzer_id, role) VALUES (?, ?, ?)`,
		m.OrgID, m.UserID, m.Role)
}

func (r *memberRepo) Get(ctx context.Context, orgID, userID string) (*storage.Membership, error) {
	var m storage.Membership
	err := r.s.db.GetContext(ctx, &m, r.s.db.Rebind(
		`SELECT organization_id, benutzer_id, role FROM member WHERE organization_id = ? AND benutzer_id = ?`),
		orgID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	return &m, err
}

func (r *memberRepo) ListByUser(ctx context.Context, userID string) ([]storage.TenantMembership, error) {
	var out []storage.TenantMembership
	err := r.s.db.SelectContext(ctx, &out, r.s.db.Rebind(
		`SELECT m.organization_id, o.name, m.role
		 FROM member m JOIN organization o ON o.id = m.organization_id
		 WHERE m.benutzer_id = ? ORDER BY o.name ASC`), userID)
	return out, err
}

// --- Invites ---

type inviteRepo struct{ s *Store }

func (r *inviteRepo) Create(ctx context.Context, i *storage.Invite) error {
	return r.s.exec(ctx, `INSERT INTO invite (id, organization_id, email, role, token, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		i.ID, i.OrgID, i.Email, i.Role, i.Token, i.CreatedAt)
}

func (r *inviteRepo) GetByToken(ctx context.Context, token string) (*storage.Invite, error) {
	if token == "" {
		return nil, storage.ErrNotFound
	}
	var i storage.Invite
	err := r.s.db.GetContext(ctx, &i, r.s.db.Rebind(
		`SELECT id, organization_id, email, role, token, created_at FROM invite WHERE token = ?`), token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	return &i, err
}

func (r *inviteRepo) Delete(ctx context.Context, id string) error {
	return r.s.exec(ctx, `DELETE FROM invite WHERE id = ?`, id)
}
