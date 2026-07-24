package postgresmodels

import (
	"time"
)

type Message struct {
	ID int `pg:"id,pk,autoincrement"`
	
	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	SenderID     []byte
	SenderIDHash string
	ChatID       []byte
	ChatIDHash   string
	MessageID    int
	BusinessConnectionIDHash string

	IsDeleted bool `pg:"is_deleted,default:false"`

	MediaGroupIDHash string `pg:"media_group_id_hash"`
	Metadata         []byte `pg:"metadata"`
	MetadataFormat   int16  `pg:"metadata_format,notnull,use_zero"`
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
