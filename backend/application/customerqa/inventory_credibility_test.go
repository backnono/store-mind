package customerqa

import (
	"testing"
	"time"

	domain "store-mind/domain/customerqa"
)

func TestComputeCredibility(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		lastVerifiedAgo time.Duration // how long ago was last_verified_at
		nilInventory    bool
		wantLevel       CredLevel
	}{
		{"10 minutes ago → high", 10 * time.Minute, false, CredHigh},
		{"29 minutes ago → high", 29 * time.Minute, false, CredHigh},
		{"30 minutes ago → high (boundary)", 30*time.Minute - time.Second, false, CredHigh},
		{"31 minutes ago → medium", 31 * time.Minute, false, CredMedium},
		{"1 hour ago → medium", time.Hour, false, CredMedium},
		{"1h59m ago → medium", 1*time.Hour + 59*time.Minute, false, CredMedium},
		{"3 hours ago → low", 3 * time.Hour, false, CredLow},
		{"23h59m ago → low", 23*time.Hour + 59*time.Minute, false, CredLow},
		{"25 hours ago → reference_only", 25 * time.Hour, false, CredReferenceOnly},
		{"nil inventory → reference_only", 0, true, CredReferenceOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inv *domain.Inventory
			if !tt.nilInventory {
				tv := now.Add(-tt.lastVerifiedAgo)
				inv = &domain.Inventory{LastVerifiedAt: &tv}
			}
			cred := ComputeCredibility(inv)
			if cred.Level != tt.wantLevel {
				t.Errorf("ComputeCredibility(%v) = %s, want %s", tt.name, cred.Level, tt.wantLevel)
			}
		})
	}
}

func TestCredibilityTag(t *testing.T) {
	now := time.Now()

	// high: 10 min ago
	tv := now.Add(-10 * time.Minute)
	inv := &domain.Inventory{LastVerifiedAt: &tv}
	tag := CredibilityTag(inv)
	if tag == "" {
		t.Error("CredibilityTag returned empty string")
	}

	// nil inventory
	tagNil := CredibilityTag(nil)
	if tagNil != string(CredReferenceOnly) {
		t.Errorf("nil inventory tag = %s, want %s", tagNil, CredReferenceOnly)
	}

	// nil last_verified_at
	invNil := &domain.Inventory{LastVerifiedAt: nil}
	tagNil2 := CredibilityTag(invNil)
	if tagNil2 != string(CredReferenceOnly) {
		t.Errorf("nil last_verified_at tag = %s, want %s", tagNil2, CredReferenceOnly)
	}
}

func TestAllCredLevels(t *testing.T) {
	levels := AllCredLevels()
	if len(levels) != 4 {
		t.Errorf("expected 4 cred levels, got %d", len(levels))
	}
	for i, c := range levels {
		if c.Level == "" || c.Label == "" || c.Color == "" {
			t.Errorf("level[%d] has empty field: %+v", i, c)
		}
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name    string
		d       time.Duration
		contain string
	}{
		{"just now", 10 * time.Second, "刚刚"},
		{"5 min", 5 * time.Minute, "分钟前"},
		{"2 hours", 2 * time.Hour, "小时前"},
		{"3 days", 72 * time.Hour, "天前"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.d)
			if result == "" {
				t.Error("formatTimeAgo returned empty string")
			}
		})
	}
}

var _ CredLevel = CredHigh
