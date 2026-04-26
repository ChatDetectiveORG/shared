package postgresmodels

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10/orm"
)

type LevelSummary struct {
	Level                 int
	NearestDecreaseAt     int64
	NearestDecreaseAmount int
}

type UserHierarchy struct {
	Level   int
	IsAdmin bool
}

func ComputeLevelSummary(purchases []UserLevels, now time.Time) LevelSummary {
	nowUnix := now.Unix()
	summary := LevelSummary{}

	for _, purchase := range purchases {
		if purchase.UntilTimestamp <= nowUnix || purchase.Level <= 0 {
			continue
		}

		summary.Level += purchase.Level
		if summary.NearestDecreaseAt == 0 || purchase.UntilTimestamp < summary.NearestDecreaseAt {
			summary.NearestDecreaseAt = purchase.UntilTimestamp
			summary.NearestDecreaseAmount = purchase.Level
			continue
		}
		if purchase.UntilTimestamp == summary.NearestDecreaseAt {
			summary.NearestDecreaseAmount += purchase.Level
		}
	}

	return summary
}

func GetUserLevelSummary(db orm.DB, userID []byte, now time.Time) (LevelSummary, *e.ErrorInfo) {
	var purchases []UserLevels
	eraw := db.Model(&purchases).
		Where("linked_user_id = ?", userID).
		Where("until_timestamp > ?", now.Unix()).
		Select()
	if eraw != nil {
		return LevelSummary{}, e.FromError(eraw, "failed to get user levels").WithSeverity(e.Notice)
	}

	return ComputeLevelSummary(purchases, now), e.Nil()
}

func IsUserAdmin(db orm.DB, userID []byte) (bool, *e.ErrorInfo) {
	count, eraw := db.Model((*Admin)(nil)).
		Where("user_id = ?", userID).
		Count()
	if eraw != nil {
		return false, e.FromError(eraw, "failed to check admin status").WithSeverity(e.Notice)
	}
	return count > 0, e.Nil()
}

func GetUserHierarchy(db orm.DB, userID []byte, now time.Time) (UserHierarchy, *e.ErrorInfo) {
	level, err := GetUserLevelSummary(db, userID, now)
	if e.IsNonNil(err) {
		return UserHierarchy{}, err
	}

	isAdmin, err := IsUserAdmin(db, userID)
	if e.IsNonNil(err) {
		return UserHierarchy{}, err
	}

	return UserHierarchy{Level: level.Level, IsAdmin: isAdmin}, e.Nil()
}

func CanReceiveNotification(receiver, actor UserHierarchy) bool {
	if receiver.IsAdmin && actor.IsAdmin {
		return true
	}
	if receiver.IsAdmin {
		return true
	}
	if actor.IsAdmin {
		return false
	}
	return receiver.Level >= actor.Level
}
