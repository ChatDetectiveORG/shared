package legal

import "testing"

func TestFromEnvTrimsAndReads(t *testing.T) {
	t.Setenv(AgreementURLEnv, " https://example.com/agreement ")
	t.Setenv(PrivacyURLEnv, "https://example.com/privacy")
	t.Setenv(ConsentURLEnv, "https://example.com/consent")
	t.Setenv(DocsVersionEnv, "2026-07-20")

	docs := FromEnv()
	if docs.AgreementURL != "https://example.com/agreement" {
		t.Fatalf("unexpected agreement url: %q", docs.AgreementURL)
	}
	if !docs.Configured() {
		t.Fatal("expected docs to be configured")
	}
}

func TestConfiguredRequiresAllFields(t *testing.T) {
	full := Docs{
		AgreementURL: "https://example.com/agreement",
		PrivacyURL:   "https://example.com/privacy",
		ConsentURL:   "https://example.com/consent",
		Version:      "v1",
	}
	if !full.Configured() {
		t.Fatal("expected full docs to be configured")
	}

	mutations := []func(Docs) Docs{
		func(d Docs) Docs { d.AgreementURL = ""; return d },
		func(d Docs) Docs { d.PrivacyURL = ""; return d },
		func(d Docs) Docs { d.ConsentURL = ""; return d },
		func(d Docs) Docs { d.Version = ""; return d },
	}
	for i, mutate := range mutations {
		if mutate(full).Configured() {
			t.Fatalf("mutation %d: expected docs to be not configured", i)
		}
	}
}
