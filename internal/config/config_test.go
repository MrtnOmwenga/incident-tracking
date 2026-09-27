package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDefaults(t *testing.T) {
	c, err := Load(env(map[string]string{"DATABASE_URL": "postgres://x"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Env != "development" || c.Addr != ":8080" || c.CheckWorkers != 8 || c.SandboxTTL.Minutes() != 120 || c.SecureCookies() {
		t.Fatalf("%+v", c)
	}
}

func TestProductionIsStrict(t *testing.T) {
	_, err := Load(env(map[string]string{
		"LIGHTHOUSE_ENV": "production", "DATABASE_URL": "postgres://x", "PUBLIC_URL": "http://status.example", "DEV_LOGIN": "true",
	}))
	if err == nil {
		t.Fatal("accepted an unsafe production config")
	}
	// Every problem is reported at once.
	for _, want := range []string{"DEV_LOGIN", "https", "OWNER_GITHUB_ID"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in: %v", want, err)
		}
	}

	c, err := Load(env(map[string]string{
		"LIGHTHOUSE_ENV": "production", "DATABASE_URL": "postgres://x", "PUBLIC_URL": "https://status.example/",
		"GITHUB_CLIENT_ID": "id", "GITHUB_CLIENT_SECRET": "secret", "OWNER_GITHUB_ID": "12345678",
	}))
	if err != nil || !c.SecureCookies() || c.PublicURL != "https://status.example" || c.OwnerGitHubID != 12345678 {
		t.Fatalf("a valid production config: %+v %v", c, err)
	}
}

func TestRejectsBadValues(t *testing.T) {
	for key, value := range map[string]string{
		"CHECK_WORKERS": "0", "SANDBOX_TTL_MINUTES": "lots", "OWNER_GITHUB_ID": "MrtnOmwenga", "PUBLIC_URL": "status.example",
		"LIGHTHOUSE_ENV": "staging", "DATABASE_URL": " ",
	} {
		vars := map[string]string{"DATABASE_URL": "postgres://x", key: value}
		if _, err := Load(env(vars)); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s=%q: %v", key, value, err)
		}
	}
}

func TestAlertSettings(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://x"}
	with := func(extra map[string]string) (Config, error) {
		vars := map[string]string{}
		for k, v := range base {
			vars[k] = v
		}
		for k, v := range extra {
			vars[k] = v
		}
		return Load(env(vars))
	}
	if c, err := with(nil); err != nil || c.AlertTo != nil {
		t.Fatalf("alerts are off by default: %+v %v", c.AlertTo, err)
	}
	_, err := with(map[string]string{"ALERT_TO": "me@example.com"})
	if err == nil || !strings.Contains(err.Error(), "SMTP_HOST") || !strings.Contains(err.Error(), "ALERT_FROM") {
		t.Fatalf("alerts need a server and a sender: %v", err)
	}
	if _, err := with(map[string]string{"ALERT_TO": "not an address", "SMTP_HOST": "smtp.example.com", "ALERT_FROM": "l@example.com"}); err == nil {
		t.Fatal("a bad recipient must be refused")
	}
	c, err := with(map[string]string{"ALERT_TO": "a@example.com, Martin <b@example.com>", "SMTP_HOST": "smtp.example.com", "ALERT_FROM": "l@example.com"})
	if err != nil || len(c.AlertTo) != 2 || c.AlertTo[1] != "b@example.com" || c.SMTPPort != 587 {
		t.Fatalf("%+v %v", c, err)
	}
}
