package postgresmodels

import (
	"time"

	tele "gopkg.in/telebot.v4"
)

type Message struct {
	ID int `pg:"id,pk,autoincrement"`
	
	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	SenderID     string
	ChatID       string
	MessageID    int
	BusinessConnectionID string

	IsDeleted bool `pg:"is_deleted,default:false"`

	Metadata *tele.Message `pg:"metadata,type:jsonb"`
}

type MessageVersion struct {
	ID int `pg:"id,pk,autoincrement"`
	
	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	MessageID int
	Message *Message `pg:"rel:has-one,fk:message_id"`

	OldMessageMetadata *tele.Message `pg:"old_message_metadata,type:jsonb"`
}
