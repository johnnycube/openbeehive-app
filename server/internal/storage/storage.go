// Package storage defines backend-independent repository interfaces.
// The SQL implementation (Postgres/MySQL/SQLite) lives in storage/sql,
// the blob implementations (MinIO/FS) in storage/blob.
package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

// --- Domain models (DB view, decoupled from protobuf) ---

type Apiary struct {
	ID        string    `db:"id"`
	OrgID     string    `db:"organization_id"`
	Name      string    `db:"name"`
	Address      string    `db:"address"`
	Lat       float64   `db:"lat"`
	Lng       float64   `db:"lng"`
	Note     string    `db:"note"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Hive struct {
	ID           string    `db:"id"`
	OrgID        string    `db:"organization_id"`
	ApiaryID   string    `db:"apiary_id"`
	Name         string    `db:"name"`
	Type          int32     `db:"type"`
	Status       int32     `db:"status"`
	Boxes       int32     `db:"boxes"`
	ColonyOrigin string    `db:"colony_origin"`
	Note        string    `db:"note"`
	QRCode       string    `db:"qr_code"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// (more Modelle: Queen, Inspection, Task, Treatment, Harvest – analog)

// User is an account on the instance (email/password onboarding). The first
// account created becomes the admin.
type User struct {
	ID            string    `db:"id"`
	Email         string    `db:"email"`
	Name          string    `db:"name"`
	OIDCSubject   string    `db:"oidc_subject"`
	PasswordHash  string    `db:"password_hash"`
	Role          string    `db:"role"` // admin | user
	EmailVerified bool      `db:"email_verified"`
	VerifyToken   string    `db:"verification_token"`
	CreatedAt     time.Time `db:"created_at"`
}

// Organization is a tenant: a collection of apiaries/hives that members share.
// A user has a personal tenant and may belong to others (e.g. a club).
type Organization struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Plan      string    `db:"plan"`
	CreatedAt time.Time `db:"created_at"`
}

// Membership links a user to a tenant with a tenant role (owner = tenant admin).
type Membership struct {
	OrgID  string `db:"organization_id"`
	UserID string `db:"benutzer_id"`
	Role   string `db:"role"` // owner | member
}

// TenantMembership is a user's membership joined with the tenant name (for UI).
type TenantMembership struct {
	OrgID string `db:"organization_id"`
	Name  string `db:"name"`
	Role  string `db:"role"`
}

// Invite lets a tenant admin invite someone to their tenant by email.
type Invite struct {
	ID        string    `db:"id"`
	OrgID     string    `db:"organization_id"`
	Email     string    `db:"email"`
	Role      string    `db:"role"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
}

// --- Repository-Interfaces ---

type UserRepo interface {
	Count(ctx context.Context) (int, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetBySubject(ctx context.Context, oidcSubject string) (*User, error)
	GetByVerifyToken(ctx context.Context, token string) (*User, error)
	Create(ctx context.Context, u *User) error
	MarkVerified(ctx context.Context, id string) error
	LinkOIDC(ctx context.Context, id, oidcSubject string) error
}

type OrgRepo interface {
	Create(ctx context.Context, o *Organization) error
	Get(ctx context.Context, id string) (*Organization, error)
}

type MemberRepo interface {
	Add(ctx context.Context, m *Membership) error
	Get(ctx context.Context, orgID, userID string) (*Membership, error)
	ListByUser(ctx context.Context, userID string) ([]TenantMembership, error)
}

type InviteRepo interface {
	Create(ctx context.Context, i *Invite) error
	GetByToken(ctx context.Context, token string) (*Invite, error)
	Delete(ctx context.Context, id string) error
}

type ApiaryRepo interface {
	Create(ctx context.Context, s *Apiary) error
	Get(ctx context.Context, orgID, id string) (*Apiary, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]Apiary, int, error)
	Update(ctx context.Context, s *Apiary) error
	Delete(ctx context.Context, orgID, id string) error
	HiveCount(ctx context.Context, orgID, apiaryID string) (int, error)
}

type HiveRepo interface {
	Create(ctx context.Context, b *Hive) error
	Get(ctx context.Context, orgID, id string) (*Hive, error)
	List(ctx context.Context, orgID, apiaryID string, limit, offset int) ([]Hive, int, error)
	Update(ctx context.Context, b *Hive) error
	Delete(ctx context.Context, orgID, id string) error
}

// Store bundles all repositories plus migrations/lifecycle.
type Store interface {
	Apiaries() ApiaryRepo
	Hives() HiveRepo
	Users() UserRepo
	Orgs() OrgRepo
	Members() MemberRepo
	Invites() InviteRepo
	Migrate(ctx context.Context) error
	Close() error
}
