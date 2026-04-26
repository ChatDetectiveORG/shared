package postgresmodels

import (
	"strconv"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	u "github.com/ChatDetectiveORG/shared/utils"
	"github.com/go-pg/pg/v10/orm"
)

const (
	MirrorStatusPending = "pending"
	MirrorStatusActive  = "active"
	MirrorStatusDeleted = "deleted"
)

type Mirror struct {
	ID int `pg:"id,pk"`

	Unique string `pg:"unique,unique"`
	Status string `pg:"status"`

	Token     []byte `pg:"token"`
	TokenHash string `pg:"token_hash,unique"`

	BotIDHash   string `pg:"bot_id_hash,unique"`
	BotUsername string `pg:"bot_username"`

	LastUsedAt *time.Time `pg:"last_used_at"`
	PaidUntil  *time.Time `pg:"paid_until"`

	OwnerID []byte        `pg:"owner_id,fk:telegram_user_id"`
	Owner   *Telegramuser `pg:"rel:has-one,fk:owner_id"`

	SourcePaymentID *int     `pg:"source_payment_id"`
	SourcePayment   *Payment `pg:"rel:has-one,fk:source_payment_id"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`
}

func NewMirror(owner *Telegramuser, token string, botID int64, botUsername string, unique string, status string, now time.Time) (*Mirror, *e.ErrorInfo) {
	if owner == nil {
		return nil, e.NewError("owner is nil", "failed to build mirror").WithSeverity(e.Notice)
	}
	encryptedToken, tokenHash, err := encryptMirrorToken(owner, token)
	if e.IsNonNil(err) {
		return nil, err
	}
	botIDHash, err := u.ToSecureHash(botID)
	if e.IsNonNil(err) {
		return nil, err
	}
	if status == "" {
		status = MirrorStatusPending
	}
	return &Mirror{
		Unique:      unique,
		Status:      status,
		Token:       encryptedToken,
		TokenHash:   tokenHash,
		BotIDHash:   botIDHash,
		BotUsername: botUsername,
		OwnerID:     owner.ID,
		Owner:       owner,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, e.Nil()
}

func (m *Mirror) DecryptToken(owner *Telegramuser) (string, *e.ErrorInfo) {
	if m == nil {
		return "", e.NewError("mirror is nil", "failed to decrypt mirror token").WithSeverity(e.Notice)
	}
	if owner == nil {
		owner = m.Owner
	}
	if owner == nil {
		return "", e.NewError("mirror owner is nil", "failed to decrypt mirror token").WithSeverity(e.Notice)
	}
	key, err := u.DecryptUserKey(owner.DataEncryptionKey)
	if e.IsNonNil(err) {
		return "", err
	}
	token, err := u.Decrypt(m.Token, key)
	if e.IsNonNil(err) {
		return "", e.FromError(err, "failed to decrypt mirror token").WithSeverity(e.Notice)
	}
	return string(token), e.Nil()
}

func (m *Mirror) SetToken(owner *Telegramuser, token string) *e.ErrorInfo {
	encryptedToken, tokenHash, err := encryptMirrorToken(owner, token)
	if e.IsNonNil(err) {
		return err
	}
	m.Token = encryptedToken
	m.TokenHash = tokenHash
	m.UpdatedAt = time.Now()
	return e.Nil()
}

func FindMirrorByID(db orm.DB, id int) (*Mirror, *e.ErrorInfo) {
	mirror := &Mirror{ID: id}
	if err := db.Model(mirror).WherePK().Relation("Owner").Select(); err != nil {
		return nil, e.FromError(err, "failed to get mirror by id").WithSeverity(e.Notice)
	}
	return mirror, e.Nil()
}

func FindMirrorByUnique(db orm.DB, unique string) (*Mirror, *e.ErrorInfo) {
	mirror := &Mirror{}
	if err := db.Model(mirror).Where("\"unique\" = ?", unique).Relation("Owner").Select(); err != nil {
		return nil, e.FromError(err, "failed to get mirror by unique").WithSeverity(e.Notice)
	}
	return mirror, e.Nil()
}

func FindActiveMirrorByUnique(db orm.DB, unique string, now time.Time) (*Mirror, *e.ErrorInfo) {
	mirror := &Mirror{}
	err := db.Model(mirror).
		Where("\"unique\" = ?", unique).
		Where("status = ?", MirrorStatusActive).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			return q.WhereOr("paid_until IS NULL").WhereOr("paid_until > ?", now), nil
		}).
		Relation("Owner").
		Select()
	if err != nil {
		return nil, e.FromError(err, "failed to get active mirror by unique").WithSeverity(e.Notice)
	}
	return mirror, e.Nil()
}

func CountActiveMirrorsForOwner(db orm.DB, ownerID []byte, now time.Time) (int, *e.ErrorInfo) {
	count, err := db.Model((*Mirror)(nil)).
		Where("owner_id = ?", ownerID).
		Where("status = ?", MirrorStatusActive).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			return q.WhereOr("paid_until IS NULL").WhereOr("paid_until > ?", now), nil
		}).
		Count()
	if err != nil {
		return 0, e.FromError(err, "failed to count owner mirrors").WithSeverity(e.Notice)
	}
	return count, e.Nil()
}

func MarkMirrorUsed(db orm.DB, mirrorID int, now time.Time) *e.ErrorInfo {
	if mirrorID == 0 {
		return e.Nil()
	}
	mirror := &Mirror{ID: mirrorID, LastUsedAt: &now, UpdatedAt: now}
	if _, err := db.Model(mirror).WherePK().Column("last_used_at", "updated_at").Update(); err != nil {
		return e.FromError(err, "failed to mark mirror used").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func ActivateMirror(db orm.DB, mirrorID int, sourcePaymentID *int, now time.Time) (*Mirror, *e.ErrorInfo) {
	mirror, err := FindMirrorByID(db, mirrorID)
	if e.IsNonNil(err) {
		return nil, err
	}
	paidUntil := now.AddDate(0, 1, 0)
	mirror.Status = MirrorStatusActive
	mirror.PaidUntil = &paidUntil
	mirror.SourcePaymentID = sourcePaymentID
	mirror.UpdatedAt = now
	if _, rawErr := db.Model(mirror).WherePK().Column("status", "paid_until", "source_payment_id", "updated_at").Update(); rawErr != nil {
		return nil, e.FromError(rawErr, "failed to activate mirror").WithSeverity(e.Notice)
	}
	return mirror, e.Nil()
}

func MirrorIDHeaderValue(id int) string {
	if id == 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func ParseMirrorID(value string) (int, *e.ErrorInfo) {
	if value == "" {
		return 0, e.Nil()
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, e.FromError(err, "failed to parse mirror id").WithSeverity(e.Notice)
	}
	return id, e.Nil()
}

func encryptMirrorToken(owner *Telegramuser, token string) ([]byte, string, *e.ErrorInfo) {
	if owner == nil {
		return nil, "", e.NewError("owner is nil", "failed to encrypt mirror token").WithSeverity(e.Notice)
	}
	key, err := u.DecryptUserKey(owner.DataEncryptionKey)
	if e.IsNonNil(err) {
		return nil, "", err
	}
	encryptedToken, err := u.Encrypt([]byte(token), key)
	if e.IsNonNil(err) {
		return nil, "", e.FromError(err, "failed to encrypt mirror token").WithSeverity(e.Notice)
	}
	tokenHash, err := u.ToSecureHash(token)
	if e.IsNonNil(err) {
		return nil, "", err
	}
	return encryptedToken, tokenHash, e.Nil()
}
