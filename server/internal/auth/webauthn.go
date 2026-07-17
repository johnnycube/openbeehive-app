package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/johnnycube/openbeehive-app/server/internal/config"
)

// WebAuthn implements passkey registration and login backed by SQL storage.
// Challenge state between begin/finish is kept in-memory keyed by a cookie.
type WebAuthn struct {
	wa       *webauthn.WebAuthn
	db       *sqlx.DB
	sessions sync.Map // challenge id -> *webauthn.SessionData
}

func NewWebAuthn(cfg *config.Config, db *sqlx.DB) (*WebAuthn, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.Auth.WebAuthn.DisplayName,
		RPID:          cfg.Auth.WebAuthn.RPID,
		RPOrigins:     cfg.Auth.WebAuthn.RPOrigins,
	})
	if err != nil {
		return nil, err
	}
	return &WebAuthn{wa: w, db: db}, nil
}

func (a *WebAuthn) Routes(mux *http.ServeMux, sessions *SessionManager) {
	mux.HandleFunc("/auth/webauthn/register/begin", a.registerBegin)
	mux.HandleFunc("/auth/webauthn/register/finish", a.registerFinish)
	mux.HandleFunc("/auth/webauthn/login/begin", a.loginBegin)
	mux.HandleFunc("/auth/webauthn/login/finish", func(w http.ResponseWriter, r *http.Request) {
		a.loginFinish(w, r, sessions)
	})
}

// --- webauthn.User ---

type waUser struct {
	id      string
	name    string
	display string
	creds   []webauthn.Credential
}

func (u *waUser) WebAuthnID() []byte                         { return []byte(u.id) }
func (u *waUser) WebAuthnName() string                       { return u.name }
func (u *waUser) WebAuthnDisplayName() string                { return u.display }
func (u *waUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

// --- store ---

func (a *WebAuthn) credentials(ctx context.Context, userID string) []webauthn.Credential {
	var rows []string
	_ = a.db.SelectContext(ctx, &rows, a.db.Rebind(
		`SELECT cred FROM webauthn_credential WHERE user_id = ?`), userID)
	out := make([]webauthn.Credential, 0, len(rows))
	for _, s := range rows {
		var c webauthn.Credential
		if json.Unmarshal([]byte(s), &c) == nil {
			out = append(out, c)
		}
	}
	return out
}

func (a *WebAuthn) userByName(ctx context.Context, name string) (*waUser, error) {
	var row struct {
		ID      string `db:"id"`
		Name    string `db:"name"`
		Display string `db:"display_name"`
	}
	if err := a.db.GetContext(ctx, &row, a.db.Rebind(
		`SELECT id, name, display_name FROM webauthn_user WHERE name = ?`), name); err != nil {
		return nil, err
	}
	return &waUser{id: row.ID, name: row.Name, display: row.Display, creds: a.credentials(ctx, row.ID)}, nil
}

func (a *WebAuthn) userByID(ctx context.Context, id string) (*waUser, error) {
	var row struct {
		ID      string `db:"id"`
		Name    string `db:"name"`
		Display string `db:"display_name"`
	}
	if err := a.db.GetContext(ctx, &row, a.db.Rebind(
		`SELECT id, name, display_name FROM webauthn_user WHERE id = ?`), id); err != nil {
		return nil, err
	}
	return &waUser{id: row.ID, name: row.Name, display: row.Display, creds: a.credentials(ctx, row.ID)}, nil
}

func (a *WebAuthn) getOrCreateUser(ctx context.Context, name string) (*waUser, error) {
	if u, err := a.userByName(ctx, name); err == nil {
		return u, nil
	}
	id := uuid.NewString()
	if _, err := a.db.ExecContext(ctx, a.db.Rebind(
		`INSERT INTO webauthn_user (id, name, display_name) VALUES (?, ?, ?)`), id, name, name); err != nil {
		return nil, err
	}
	return &waUser{id: id, name: name, display: name}, nil
}

func (a *WebAuthn) addCredential(ctx context.Context, userID string, cred *webauthn.Credential) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, a.db.Rebind(
		`INSERT INTO webauthn_credential (id, user_id, cred) VALUES (?, ?, ?)`),
		base64.RawURLEncoding.EncodeToString(cred.ID), userID, string(raw))
	return err
}

// --- challenge session (begin -> finish) ---

const waChallengeCookie = "obh_wa"

func (a *WebAuthn) storeChallenge(w http.ResponseWriter, s *webauthn.SessionData) {
	id := randToken()
	a.sessions.Store(id, s)
	http.SetCookie(w, &http.Cookie{
		Name: waChallengeCookie, Value: id, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: true, MaxAge: 300,
	})
}

func (a *WebAuthn) loadChallenge(r *http.Request) (*webauthn.SessionData, bool) {
	ck, err := r.Cookie(waChallengeCookie)
	if err != nil {
		return nil, false
	}
	v, ok := a.sessions.LoadAndDelete(ck.Value)
	if !ok {
		return nil, false
	}
	return v.(*webauthn.SessionData), true
}

// --- handlers ---

func (a *WebAuthn) registerBegin(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("username")
	if name == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	user, err := a.getOrCreateUser(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	options, session, err := a.wa.BeginRegistration(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.storeChallenge(w, session)
	writeJSON(w, options)
}

func (a *WebAuthn) registerFinish(w http.ResponseWriter, r *http.Request) {
	session, ok := a.loadChallenge(r)
	if !ok {
		http.Error(w, "no registration in progress", http.StatusBadRequest)
		return
	}
	user, err := a.userByID(r.Context(), string(session.UserID))
	if err != nil {
		http.Error(w, "unknown user", http.StatusBadRequest)
		return
	}
	cred, err := a.wa.FinishRegistration(user, *session, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.addCredential(r.Context(), user.id, cred); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *WebAuthn) loginBegin(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("username")
	user, err := a.userByName(r.Context(), name)
	if err != nil {
		http.Error(w, "unknown user", http.StatusUnauthorized)
		return
	}
	options, session, err := a.wa.BeginLogin(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.storeChallenge(w, session)
	writeJSON(w, options)
}

func (a *WebAuthn) loginFinish(w http.ResponseWriter, r *http.Request, sessions *SessionManager) {
	session, ok := a.loadChallenge(r)
	if !ok {
		http.Error(w, "no login in progress", http.StatusBadRequest)
		return
	}
	user, err := a.userByID(r.Context(), string(session.UserID))
	if err != nil {
		http.Error(w, "unknown user", http.StatusUnauthorized)
		return
	}
	if _, err := a.wa.FinishLogin(user, *session, r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	token, err := sessions.Issue(Identity{UserID: user.id, OrgID: user.id, Email: user.name, Role: "owner"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sessions.SetCookie(w, token)
	writeJSON(w, map[string]string{"token": token})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
