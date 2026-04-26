package postgresmodels

import "time"

const (
	ReferralBonusMoney    = "money"
	ReferralBonusDiscount = "discount"
	ReferralBonusLevels   = "levels"
)

type UserSettings struct {
	ID           int           `pg:"id,pk,autoincrement"`
	LinkedUserID []byte        `pg:",fk:telegram_user_id,unique,pk"`
	LinkedUser   *Telegramuser `pg:"rel:has-one,fk:linked_user_id"`

	NotifyAboutDeletedMessages bool   `pg:"notify_about_deleted_messages,default:true"`
	NotifyAboutEditedMessages  bool   `pg:"notify_about_edited_messages,default:true"`
	SaveSelfDistructingMedia   bool   `pg:"save_self_destructing_media,default:true"`
	ExtendedChatExport         bool   `pg:"extended_chat_export,default:false"`
	ReferralBonusPreference    string `pg:"referral_bonus_preference,default:'discount'"`
}

type UserLevels struct {
	ID int `pg:"id,pk"`

	LinkedUserID []byte        `pg:"linked_user_id,fk:telegram_user_id"`
	LinkedUser   *Telegramuser `pg:"rel:has-one,fk:linked_user_id"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	Level           int   `pg:"level"`
	UntilTimestamp  int64 `pg:"until_timestamp"`
	SourcePaymentID *int  `pg:"source_payment_id,unique"`
}

type Admin struct {
	ID             int `pg:"id,pk,autoincrement"`
	UserID         []byte
	User           *Telegramuser `pg:"rel:has-one,fk:user_id"`
	PermissionsLvl int
	Note           string
}
