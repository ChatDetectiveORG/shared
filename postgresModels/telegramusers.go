package postgresmodels

import (
	"crypto/rand"
	"encoding/json"
	"strconv"
	"time"

	u "github.com/ChatDetectiveORG/shared/utils"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	tele "gopkg.in/telebot.v4"
)

type Telegramuser struct {
	ID                       []byte `pg:"id,pk"`
	BusinessConnectionIDHash string

	DataEncryptionKey []byte

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	Fullname []byte
	Username []byte

	Metadata []byte `pg:"metadata"`
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
	if len(t.DataEncryptionKey) == 0 {
		return e.NewError("data encryption key is not set", "data encryption key is not set").WithSeverity(e.Notice)
	}

	key, err := u.DecryptUserKey(t.DataEncryptionKey)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to decrypt data encryption key").WithSeverity(e.Notice)
	}

	idEncr, err := u.Encrypt([]byte(strconv.FormatInt(userID, 10)), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user id").WithSeverity(e.Notice)
	}

	errUnwrapped := db.Model(t).Where("id = ?", idEncr).Select()
	if e.IsNonNil(errUnwrapped) {
		return e.FromError(errUnwrapped, "error getting telegram user")
	}

	return e.Nil()
}

func (t *Telegramuser) GetOrCreate(tx *pg.Tx, tguser *tele.User) *e.ErrorInfo {
	err := t.GetByTelegramID(tx, tguser.ID)
	if e.IsNil(err) {
		return nil
	}
	err = e.Nil()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return e.FromError(err, "failed to read full random reader").WithSeverity(e.Critical)
	}

	if e.IsNonNil(err) {
		return e.FromError(err, "failed to generate user secret key").WithSeverity(e.Notice)
	}

	encryptedID, err := u.Encrypt([]byte(strconv.FormatInt(tguser.ID, 10)), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user id").WithSeverity(e.Notice)
	}

	encryptedFullname, err := u.Encrypt([]byte(tguser.FirstName+" "+tguser.LastName), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user fullname").WithSeverity(e.Notice)
	}

	encryptedUsername, err := u.Encrypt([]byte(tguser.Username), key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user username").WithSeverity(e.Notice)
	}

	jsonMetadata, eraw := json.Marshal(tguser)
	if e.IsNonNil(eraw) {
		return e.FromError(eraw, "failed to encrypt telegram user metadata").WithSeverity(e.Notice)
	}

	encryptedMetadata, err := u.Encrypt(jsonMetadata, key)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user metadata").WithSeverity(e.Notice)
	}

	masterKey, err := u.GetMasterkey()
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to get master key").WithSeverity(e.Critical)
	}

	encryptedKey, err := u.Encrypt(key, masterKey)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt data encryption key").WithSeverity(e.Critical)
	}

	user := &Telegramuser{
		ID:                encryptedID,
		Fullname:          encryptedFullname,
		Username:          encryptedUsername,
		Metadata:          encryptedMetadata,
		DataEncryptionKey: encryptedKey,
	}

	settings := &UserSettings{
		LinkedUserID: encryptedID,
		LinkedUser:   user,
	}

	_, errUnwrapped := tx.Model(user).
		OnConflict("(id) DO UPDATE").
		Set("fullname = EXCLUDED.fullname, username = EXCLUDED.username, is_bot_peer = EXCLUDED.is_bot_peer, metadata = EXCLUDED.metadata").
		Insert()
	if e.IsNonNil(errUnwrapped) {
		return e.FromError(errUnwrapped, "error creating telegram user")
	}

	_, errUnwrapped = tx.Model(settings).
		OnConflict("(linked_user_id) DO NOTHING").
		Insert()
	if e.IsNonNil(errUnwrapped) {
		tx.Rollback()
		return e.FromError(errUnwrapped, "error creating user settings")
	}

	return e.Nil()
}
