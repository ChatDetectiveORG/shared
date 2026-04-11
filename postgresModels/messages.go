package postgresmodels

import (
	"time"
)

type Message struct {
	ID int `pg:"id,pk,autoincrement"`
	
	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	SenderID     []byte
	ChatID       []byte
	MessageID    int
	BusinessConnectionIDHash string

	IsDeleted bool `pg:"is_deleted,default:false"`

	Metadata []byte `pg:"metadata"`
}

// For extended chat export
// Для базового функционала не нужно сохранение всех версий сообщения
type MessageVersion struct {
	ID int `pg:"id,pk,autoincrement"`
	
	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	MessageID int
	Message *Message `pg:"rel:has-one,fk:message_id"`

	OldMessageMetadata []byte `pg:"old_message_metadata"`
}
