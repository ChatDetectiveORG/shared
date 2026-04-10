package postgresmodels

import (
	u "github.com/ChatDetectiveORG/shared/utils"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	tele "gopkg.in/telebot.v4"
)

type Telegramuser struct {
	ID string `pg:"id,pk"`
	BusinessConnectionID string

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	Fullname string
	Username string

	Metadata *tele.User `pg:"metadata,type:jsonb"`
}

func (t *Telegramuser) get(db orm.DB, userID int64) error {
	err := db.Model(t).Where("id = ?", u.Int64ToHash(userID)).Select()
	if e.IsNonNil(err) {
		return 0, err
	}

	id, err := u.Decrypt([]byte(t.ID), masterKey)
	if e.IsNonNil(err) {
		return 0, e.FromError(err, "failed to decrypt telegram user id").WithSeverity(e.Notice)
	}

	idInt, err := strconv.ParseInt(string(id), 10, 64)
	if e.IsNonNil(err) {
		return 0, e.FromError(err, "failed to parse telegram user id").WithSeverity(e.Notice)
	}

	return idInt, e.Nil()
}

func (t *Telegramuser) get(db orm.DB, userID int64) e.ErrorInfo {
	masterKey, err := u.GetMasterkey()
	if e.IsNonNil(err) {
		return err
	}

	idEncr, err := u.Encrypt([]byte(strconv.FormatInt(userID, 10)), masterKey)
	if e.IsNonNil(err) {
		return e.FromError(err, "failed to encrypt telegram user id").WithSeverity(e.Notice)
	}

	errUnwrapped := db.Model(t).Where("id = ?", idEncr).Select()
	if e.IsNonNil(errUnwrapped) {
		return e.FromError(errUnwrapped, "error getting telegram user")
	}

	return e.Nil()
}

func (t *Telegramuser) GetOrCreate(tx *pg.Tx, tguser *tele.User) error {
	err := t.get(tx, tguser.ID)
	if e.IsNil(err) {
		return nil
	}

	user := &Telegramuser{
		ID:       u.Int64ToHash(tguser.ID),
		Fullname: tguser.FirstName + " " + tguser.LastName,
		Username: tguser.Username,
		Metadata: tguser,
	}

	settings := &UserSettings{
		LinkedUserID: user.ID,
		LinkedUser: user,
	}

	_, err = tx.Model(user).
		OnConflict("(id) DO UPDATE").
		Set("fullname = EXCLUDED.fullname, username = EXCLUDED.username, is_bot_peer = EXCLUDED.is_bot_peer, metadata = EXCLUDED.metadata").
		Insert()
	if e.IsNonNil(err) {
		return e.FromError(err, "error creating telegram user")
	}

	_, err = tx.Model(settings).
		OnConflict("(linked_user_id) DO NOTHING").
		Insert()
	if e.IsNonNil(err) {
		tx.Rollback()
		return e.FromError(err, "error creating user settings")
	}

	return e.Nil()
}
