package postgresmodels

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10/orm"
)

// UserConsent records the fact that a user accepted a specific version of the legal
// documents (user agreement + personal data consent). A new row is required whenever
// LEGAL_DOCS_VERSION changes.
type UserConsent struct {
	ID int `pg:"id,pk"`

	LinkedUserID []byte        `pg:"linked_user_id,notnull,unique:user_consent_version"`
	LinkedUser   *Telegramuser `pg:"rel:has-one,fk:linked_user_id"`

	// DocsVersion is the LEGAL_DOCS_VERSION active at the moment of consent.
	DocsVersion string `pg:"docs_version,notnull,unique:user_consent_version"`

	// Source describes where the consent was given (e.g. "telegram_start").
	Source string `pg:"source,notnull"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
}

// HasUserConsent reports whether the user already accepted the given docs version.
func HasUserConsent(db orm.DB, userID []byte, docsVersion string) (bool, *e.ErrorInfo) {
	count, err := db.Model((*UserConsent)(nil)).
		Where("linked_user_id = ?", userID).
		Where("docs_version = ?", docsVersion).
		Count()
	if err != nil {
		return false, e.FromError(err, "failed to check user consent").WithSeverity(e.Notice)
	}
	return count > 0, e.Nil()
}

// RecordUserConsent stores the consent fact idempotently: repeated accepts of the same
// version do not create duplicates.
func RecordUserConsent(db orm.DB, userID []byte, docsVersion string, source string, now time.Time) *e.ErrorInfo {
	if docsVersion == "" {
		return e.NewError("docs version is empty", "failed to record user consent").WithSeverity(e.Notice)
	}
	consent := &UserConsent{
		LinkedUserID: userID,
		DocsVersion:  docsVersion,
		Source:       source,
		CreatedAt:    now,
	}
	if _, err := db.Model(consent).
		OnConflict("(linked_user_id, docs_version) DO NOTHING").
		Insert(); err != nil {
		return e.FromError(err, "failed to record user consent").WithSeverity(e.Notice)
	}
	return e.Nil()
}
