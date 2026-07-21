package postgresmodels

import "time"

// Модель, показывающая потенуиально знакомых друг другу пользователей
// Нужно для того, чтобы показывать не-премиум пользователям, в чатах с кем они могут смотреть изменённые и удалённые особщения
type UserRelations struct {
	ID int `pg:"id,pk"`

	CreatedAt time.Time `pg:"created_at,default:now()"`

	FirstUserIDHash string
	FirstUser   *Telegramuser `pg:"rel:has-one,fk:first_user_id_hash"`

	SecondUserIDHash string
	SecondUser   *Telegramuser `pg:"rel:has-one,fk:second_user_id_hash"`
}
