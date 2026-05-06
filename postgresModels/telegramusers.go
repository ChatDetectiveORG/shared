package postgresmodels

import (
	"crypto/rand"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	u "github.com/ChatDetectiveORG/shared/utils"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	tele "gopkg.in/telebot.v4"
)

type Telegramuser struct {
	ID                       []byte `pg:"id,pk"`
	IDHash                   string `pg:"id_hash"`
	BusinessConnectionIDHash string

	DataEncryptionKey []byte

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	Fullname []byte
	Username []byte

	Metadata []byte `pg:"metadata"`

	ReferralCode string `pg:"referral_code,unique,type:varchar(8),notnull"`
	Settings *UserSettings `pg:"rel:has-one,fk:id,join_fk:linked_user_id"`
}

func (t *Telegramuser) GetFullName() (string, *e.ErrorInfo) {
	key, err := u.DecryptUserKey(t.DataEncryptionKey)
	if e.IsNonNil(err) {
		return "", e.FromError(err, "failed to decrypt data encryption key").WithSeverity(e.Notice)
	}

	fullname, err := u.Decrypt(t.Fullname, key)
	if e.IsNonNil(err) {
		return "", e.FromError(err, "failed to decrypt telegram user fullname").WithSeverity(e.Notice)
	}

	return string(fullname), e.Nil()
}

func (t *Telegramuser) GetTgId() (int64, *e.ErrorInfo) {
	key, err := u.DecryptUserKey(t.DataEncryptionKey)
	if e.IsNonNil(err) {
		return 0, e.FromError(err, "failed to decrypt data encryption key").WithSeverity(e.Notice)
	}

	id, err := u.Decrypt(t.ID, key)
	if e.IsNonNil(err) {
		return 0, e.FromError(err, "failed to decrypt telegram user id").WithSeverity(e.Notice)
	}

	idInt, errUnwrapped := strconv.ParseInt(string(id), 10, 64)
	if e.IsNonNil(errUnwrapped) {
		return 0, e.FromError(errUnwrapped, "failed to parse telegram user id").WithSeverity(e.Notice)
	}

	return idInt, e.Nil()
}

func (t *Telegramuser) GetByTelegramID(db orm.DB, userID int64) *e.ErrorInfo {
	idHash, err := u.ToSecureHash(userID)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to get secure hash").WithSeverity(e.Critical)
	}

	errUnwrapped := db.Model(t).Where("id_hash = ?", idHash).Select()
	if e.IsNonNil(errUnwrapped) {
		return e.FromError(errUnwrapped, "error getting telegram user")
	}

	return e.Nil()
}

// UpdateUserData re-encrypts and persists the user's fullname, username, and metadata.
// Call this after GetByTelegramID to refresh stale profile data.
func (t *Telegramuser) UpdateUserData(db orm.DB, tguser *tele.User) *e.ErrorInfo {
	key, err := u.DecryptUserKey(t.DataEncryptionKey)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to decrypt user key for update").WithSeverity(e.Notice)
	}

	encryptedFullname, err := u.Encrypt([]byte(tguser.FirstName+" "+tguser.LastName), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt fullname on update").WithSeverity(e.Notice)
	}

	encryptedUsername, err := u.Encrypt([]byte(tguser.Username), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt username on update").WithSeverity(e.Notice)
	}

	jsonMetadata, eraw := json.Marshal(tguser)
	if eraw != nil {
		return e.FromError(eraw, "failed to marshal user metadata on update").WithSeverity(e.Notice)
	}

	encryptedMetadata, err := u.Encrypt(jsonMetadata, key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt metadata on update").WithSeverity(e.Notice)
	}

	t.Fullname = encryptedFullname
	t.Username = encryptedUsername
	t.Metadata = encryptedMetadata
	t.UpdatedAt = time.Now()

	_, eraw = db.Model(t).WherePK().Column("fullname", "username", "metadata", "updated_at").Update()
	if eraw != nil {
		return e.FromError(eraw, "failed to update telegram user data").WithSeverity(e.Notice)
	}

	return e.Nil()
}

// GetOrCreate loads the user by Telegram id or inserts a new row. The returned
// created flag is true only when a new row was inserted in this call (not when
// the user already existed).
func (t *Telegramuser) GetOrCreate(tx *pg.Tx, tguser *tele.User) (created bool, err *e.ErrorInfo) {
	err = t.GetByTelegramID(tx, tguser.ID)
	if e.IsNil(err) {
		return false, nil
	}
	err = e.Nil()

	key := make([]byte, 32)
	if _, rerr := rand.Read(key); rerr != nil {
		return false, e.FromError(rerr, "failed to read full random reader").WithSeverity(e.Critical)
	}

	encryptedID, err := u.Encrypt([]byte(strconv.FormatInt(tguser.ID, 10)), key)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to encrypt telegram user id").WithSeverity(e.Notice)
	}

	encryptedFullname, err := u.Encrypt([]byte(tguser.FirstName+" "+tguser.LastName), key)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to encrypt telegram user fullname").WithSeverity(e.Notice)
	}

	encryptedUsername, err := u.Encrypt([]byte(tguser.Username), key)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to encrypt telegram user username").WithSeverity(e.Notice)
	}

	jsonMetadata, eraw := json.Marshal(tguser)
	if e.IsNonNil(eraw) {
		return false, e.FromError(eraw, "failed to encrypt telegram user metadata").WithSeverity(e.Notice)
	}

	encryptedMetadata, err := u.Encrypt(jsonMetadata, key)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to encrypt telegram user metadata").WithSeverity(e.Notice)
	}

	masterKey, err := u.GetMasterkey()
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to get master key").WithSeverity(e.Critical)
	}

	encryptedKey, err := u.Encrypt(key, masterKey)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to encrypt data encryption key").WithSeverity(e.Critical)
	}

	idHash, err := u.ToSecureHash(tguser.ID)
	if e.IsNonNil(err) {
		return false, e.FromError(err, "failed to get secure hash").WithSeverity(e.Critical)
	}

	referralCode, eRaw := u.GenerateReferralCode(8)
	if e.IsNonNil(eRaw) {
		return false, e.FromError(eRaw, "failed to generate referral code").WithSeverity(e.Critical)
	}

	user := &Telegramuser{
		ID:                encryptedID,
		IDHash:            idHash,
		Fullname:          encryptedFullname,
		Username:          encryptedUsername,
		Metadata:          encryptedMetadata,
		DataEncryptionKey: encryptedKey,
		ReferralCode:      referralCode,
	}

	settings := &UserSettings{
		LinkedUserID: encryptedID,
	}

	for {
		_, errUnwrapped := tx.Model(user).Insert()
		if e.IsNonNil(errUnwrapped) && !strings.Contains(errUnwrapped.Error(), "duplicate key value violates unique constraint") {
			return false, e.FromError(errUnwrapped, "error creating telegram user")
		}

		if e.IsNil(errUnwrapped) {
			break
		}

		referralCode, eRaw = u.GenerateReferralCode(8)
		if e.IsNonNil(eRaw) {
			return false, e.FromError(eRaw, "failed to generate referral code").WithSeverity(e.Critical)
		}

		user.ReferralCode = referralCode
	}

	*t = *user

	_, errUnwrapped := tx.Model(settings).
		OnConflict("(linked_user_id) DO NOTHING").
		Insert()
	if e.IsNonNil(errUnwrapped) {
		tx.Rollback()
		return false, e.FromError(errUnwrapped, "error creating user settings")
	}

	return true, e.Nil()
}
