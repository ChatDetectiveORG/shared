package postgresmodels

type UserSettings struct {
	ID                                       int           `pg:"id,pk,autoincrement"`
	LinkedUserID                             []byte        `pg:",fk:telegram_user_id,unique,pk"`
	LinkedUser                               *Telegramuser `pg:"rel:has-one,fk:linked_user_id"`

	NotifyAboutDeletedMessages               bool          `pg:"notify_about_deleted_messages,default:true"`
	NotifyAboutEditedMessages                bool          `pg:"notify_about_edited_messages,default:true"`
	SaveSelfDistructingMedia                 bool          `pg:"save_self_destructing_media,default:true"`
	ExtendedChatExport                       bool          `pg:"extended_chat_export,default:false"`
}

type UserLevels struct {
	ID             int
	LinkedUserID   []byte
	LinkedUser     *Telegramuser `pg:"rel:has-one,fk:linked_user_id"`
	Level          int
	UntilTimestamp int64
}

type Admin struct {
	ID             int           `pg:"id,pk,autoincrement"`
	UserID         []byte
	User           *Telegramuser `pg:"rel:has-one,fk:user_id"`
	PermissionsLvl int           
	Note           string
}
