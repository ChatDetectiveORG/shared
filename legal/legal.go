// Package legal provides configuration for the legal consent gate shown on /start.
//
// The actual documents (user agreement, personal data policy, consent) live outside
// the bot (see ProjectDocs/legal). The bot only links to their published versions and
// records the fact of consent per document version.
package legal

import (
	"os"
	"strings"
)

// Environment variable names for legal document configuration.
const (
	AgreementURLEnv = "LEGAL_AGREEMENT_URL"
	PrivacyURLEnv   = "LEGAL_PRIVACY_URL"
	ConsentURLEnv   = "LEGAL_CONSENT_URL"
	DocsVersionEnv  = "LEGAL_DOCS_VERSION"
)

// ConsentSourceStart identifies consent given via the /start flow.
const ConsentSourceStart = "telegram_start"

// Docs describes the published legal documents the user must accept.
type Docs struct {
	AgreementURL string
	PrivacyURL   string
	ConsentURL   string
	Version      string
}

// FromEnv reads the legal document configuration from environment variables.
func FromEnv() Docs {
	return Docs{
		AgreementURL: strings.TrimSpace(os.Getenv(AgreementURLEnv)),
		PrivacyURL:   strings.TrimSpace(os.Getenv(PrivacyURLEnv)),
		ConsentURL:   strings.TrimSpace(os.Getenv(ConsentURLEnv)),
		Version:      strings.TrimSpace(os.Getenv(DocsVersionEnv)),
	}
}

// Configured reports whether the consent gate can be enforced. All four values are
// required: without published URLs and a version there is nothing valid to accept.
// When not configured the gate is skipped (dev environments); production values
// are validated at the Helm level.
func (d Docs) Configured() bool {
	return d.AgreementURL != "" && d.PrivacyURL != "" && d.ConsentURL != "" && d.Version != ""
}
