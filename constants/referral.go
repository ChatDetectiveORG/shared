package constants

const (
	// REFERRAL_MONEY_GIVEN_AFTER_SECS is how long the invited user must stay connected
	// before the money reward is credited (MVP: 24 hours).
	REFERRAL_MONEY_GIVEN_AFTER_SECS = 60 * 60 * 24
)

// Referral bonus thresholds
const (
	// ReferralBonusThresholdLevels is how many referred users unlock one bonus level grant.
	ReferralBonusThresholdLevels = 2
)
