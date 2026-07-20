package postgresmodels

import "testing"

func TestMirrorAlreadyActivatedBy(t *testing.T) {
	paymentID := 42
	otherPaymentID := 43

	cases := []struct {
		name            string
		mirror          *Mirror
		sourcePaymentID *int
		want            bool
	}{
		{
			name:            "nil mirror",
			mirror:          nil,
			sourcePaymentID: &paymentID,
			want:            false,
		},
		{
			name:            "nil source payment",
			mirror:          &Mirror{Status: MirrorStatusActive, SourcePaymentID: &paymentID},
			sourcePaymentID: nil,
			want:            false,
		},
		{
			name:            "active mirror activated by same payment",
			mirror:          &Mirror{Status: MirrorStatusActive, SourcePaymentID: &paymentID},
			sourcePaymentID: &paymentID,
			want:            true,
		},
		{
			name:            "active mirror activated by another payment",
			mirror:          &Mirror{Status: MirrorStatusActive, SourcePaymentID: &otherPaymentID},
			sourcePaymentID: &paymentID,
			want:            false,
		},
		{
			name:            "pending mirror is not treated as activated",
			mirror:          &Mirror{Status: MirrorStatusPending, SourcePaymentID: &paymentID},
			sourcePaymentID: &paymentID,
			want:            false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MirrorAlreadyActivatedBy(c.mirror, c.sourcePaymentID); got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
