package levelmanagement

import (
	"slices"
	"time"

	"github.com/ChatDetectiveORG/shared/constants"
	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/go-pg/pg/v10"
)

// Takes non-considered yet referral relations and adds new bonus levels accoardingly to the current threshold
// Old levels stay untouched even if threshold risen
// Takes: transaction, untracked relations (referrals that were not considered yet), level recipient user ID
// Returns: number of levels added, error
func RecountLevels(tx *pg.Tx, untrackedRalations []models.Referral, levelRecipientUserID []byte) (int, *e.ErrorInfo) {
	var levelsAdded int
	threshold := constants.ReferralBonusThresholdLevels
	now := time.Now()
	defaultBonusEnd := now.Add(time.Duration(constants.ReferralLevelsDurationSec) * time.Second).Unix()

	for i := 0; i+threshold <= len(untrackedRalations); i += threshold {
		addedRelationsDurations := make([]int64, 0, threshold)
		addedRelationsIDs := make([]int, 0, threshold)

		for j := i; j < i+threshold; j++ {
			ref := untrackedRalations[j]
			u := ref.ActualUntil.Unix()
			if ref.ActualUntil.IsZero() || u <= 0 {
				u = defaultBonusEnd
			}
			addedRelationsDurations = append(addedRelationsDurations, u)
			addedRelationsIDs = append(addedRelationsIDs, ref.ID)
		}

		newLevel := models.UserLevels{
			LinkedUserID:      levelRecipientUserID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			Level:             1,
			UntilTimestamp:    slices.Min(addedRelationsDurations),
			IsReferralBonus:   true,
			LinkedReferralIDs: addedRelationsIDs,
		}

		_, eRaw := tx.Model(&newLevel).Insert()
		if e.IsNonNil(eRaw) {
			return levelsAdded, e.Wrap(eRaw)
		}

		levelsAdded += 1
	}

	return levelsAdded, e.Nil()
}

// Gets untracked relations (referrals that were not considered yet)
// Takes: transaction, invitor user ID, referral
// Returns: untracked relations, error
func GetUntrackedRelations(tx *pg.Tx, invitorID []byte) ([]models.Referral, *e.ErrorInfo) {
	var untrackedRalations []models.Referral
	err := e.Wrap(tx.Model(&untrackedRalations).
		Where("invitor_id = ?", invitorID).
		Where("id NOT IN (SELECT unnest(linked_referral_ids) FROM user_levels WHERE linked_user_id = ?)", invitorID).
		Order("actual_until ASC").
		Select(),
	)
	if e.IsNonNil(err) {
		return nil, err
	}

	return untrackedRalations, e.Nil()
}
