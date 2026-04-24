package postgresmodels

import "time"

// Модель, показывающая потенуиально знакомых друг другу пользователей
// Нужно для того, чтобы показывать не-премиум пользователям, в чатах с кем они могут смотреть изменённые и удалённые особщения
type UsersInConact struct {
	ID int `pg:"id,pk"`

	CreatedAt time.Time `pg:"created_at,default:now()"`

	BotUserID []byte
	BotUser *Telegramuser `pg:"rel:has-one,fk:first_user_id"`

	InterlocutorUserID []byte
	InterlocutorUser *Telegramuser `pg:"rel:has-one,fk:interlocutor_user_id"`
}
