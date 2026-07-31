package config

import (
	"strings"
	"testing"
)

// clearBeehiveEnv guards against BEEHIVE_* leaking in from the host shell.
func clearBeehiveEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{"DEPLOYMENT_PROFILE", "PASSWORD_AUTH", "DEMO",
		"ADMIN_EMAIL", "ADMIN_PASSWORD", "REGISTRATION", "DEMO_EMAIL", "DEMO_AUTOLOGIN"} {
		t.Setenv(envPrefix+v, "")
	}
}

func TestLoadRequiresAdminWithPasswordAuth(t *testing.T) {
	clearBeehiveEnv(t)
	t.Setenv(envPrefix+"DEPLOYMENT_PROFILE", "selfhost")
	t.Setenv(envPrefix+"PASSWORD_AUTH", "true")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "BEEHIVE_ADMIN_EMAIL") {
		t.Fatalf("want missing-admin error, got %v", err)
	}

	t.Setenv(envPrefix+"ADMIN_EMAIL", "admin@example.org")
	t.Setenv(envPrefix+"ADMIN_PASSWORD", "short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "8 characters") {
		t.Fatalf("want short-password error, got %v", err)
	}

	t.Setenv(envPrefix+"ADMIN_PASSWORD", "password123")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("valid admin config: %v", err)
	}
	if cfg.Auth.AdminEmail != "admin@example.org" {
		t.Fatalf("admin email = %q", cfg.Auth.AdminEmail)
	}
}

func TestLoadDemoImpliesAdminRequirement(t *testing.T) {
	clearBeehiveEnv(t)
	t.Setenv(envPrefix+"DEPLOYMENT_PROFILE", "selfhost")
	t.Setenv(envPrefix+"DEMO", "true") // implies password auth

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "BEEHIVE_ADMIN_EMAIL") {
		t.Fatalf("want missing-admin error, got %v", err)
	}

	// The admin must not shadow the demo account.
	t.Setenv(envPrefix+"ADMIN_EMAIL", "demo@app.openbeehive.org")
	t.Setenv(envPrefix+"ADMIN_PASSWORD", "password123")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "demo account") {
		t.Fatalf("want admin-is-demo error, got %v", err)
	}
}

func TestLoadNoAuthNeedsNoAdmin(t *testing.T) {
	clearBeehiveEnv(t)
	t.Setenv(envPrefix+"DEPLOYMENT_PROFILE", "selfhost")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("single-user selfhost must not require an admin: %v", err)
	}
	if cfg.Auth.PasswordEnabled {
		t.Fatalf("selfhost default should have password auth off")
	}
}
