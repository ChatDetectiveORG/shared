package postgresmodels

import (
	"testing"
	"time"
)

func TestComputeLevelSummarySumsActivePurchases(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	summary := ComputeLevelSummary([]UserLevels{
		{Level: 1, UntilTimestamp: now.Add(time.Hour).Unix()},
		{Level: 4, UntilTimestamp: now.Add(7 * 24 * time.Hour).Unix()},
		{Level: 10, UntilTimestamp: now.Add(-time.Hour).Unix()},
		{Level: 2, UntilTimestamp: now.Add(time.Hour).Unix()},
	}, now)

	if summary.Level != 7 {
		t.Fatalf("expected total level 7, got %d", summary.Level)
	}
	if summary.NearestDecreaseAt != now.Add(time.Hour).Unix() {
		t.Fatalf("unexpected nearest decrease timestamp: %d", summary.NearestDecreaseAt)
	}
	if summary.NearestDecreaseAmount != 3 {
		t.Fatalf("expected nearest decrease amount 3, got %d", summary.NearestDecreaseAmount)
	}
}

func TestCanReceiveNotificationComparesAdminFirst(t *testing.T) {
	admin := UserHierarchy{Level: -1000, IsAdmin: true}
	regularHigh := UserHierarchy{Level: 100000}
	regularLow := UserHierarchy{Level: 1}

	if !CanReceiveNotification(admin, regularHigh) {
		t.Fatal("admin should outrank regular user regardless of numeric level")
	}
	if CanReceiveNotification(regularHigh, admin) {
		t.Fatal("regular user should not outrank admin")
	}
	if !CanReceiveNotification(admin, UserHierarchy{Level: 100000, IsAdmin: true}) {
		t.Fatal("admins should be equal between each other")
	}
	if !CanReceiveNotification(regularHigh, regularLow) {
		t.Fatal("higher regular level should receive notification")
	}
	if CanReceiveNotification(regularLow, regularHigh) {
		t.Fatal("lower regular level should not receive notification")
	}
}
