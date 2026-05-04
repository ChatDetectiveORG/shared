package postgresmodels

import "time"

const (
	ReferralBonusMoney    = "money"
	ReferralBonusLevels   = "levels"
)

type Referral struct {
	ID int `pg:"id,pk"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`
	ActualUntil time.Time `pg:"actual_until"`

	InvitorID []byte        `pg:"invitor_id"`
	Invitor   *Telegramuser `pg:"rel:has-one,fk:invitor_id"`

	InvitedUserID []byte        `pg:"invited_user_id"`
	InvitedUser   *Telegramuser `pg:"rel:has-one,fk:invited_user_id"`

	FixedRewardType string `pg:"fixed_reward_type"`
	// Вознаграждение во внутркнней валюте бота
	FixedMoneyReward int `pg:"fixed_money_reward,default:0"` // Скидки и уровни считаются динамически
}
