package main

import "strings"

const (
	defaultMemberCategoryMatch = "member"
	defaultHighTier            = "Gold"
	defaultSecondTier          = "Silver"
)

// LandscapeConfig holds configuration for the landscape being served.
type LandscapeConfig struct {
	Name        string // e.g. "CNCF"
	Description string // e.g. "Cloud Native Computing Foundation"
	// MemberCategoryMatch is the case-insensitive substring of an item's
	// category that marks it as a member (default "member").
	MemberCategoryMatch string
	// Tiers names the membership tiers behind the tier-specific membership
	// metrics: Tiers[0] backs gold_members_* and Tiers[1] backs silver_members_*.
	// Any other tier remains free-form for query_members/funding_analysis.
	// Default: ["Gold", "Silver"].
	Tiers []string
}

func defaultConfig() LandscapeConfig {
	return LandscapeConfig{
		Name:                "CNCF",
		Description:         "Cloud Native Computing Foundation",
		MemberCategoryMatch: defaultMemberCategoryMatch,
		Tiers:               []string{defaultHighTier, defaultSecondTier},
	}
}

// metricTiers returns the tiers behind the "gold" and "silver" membership
// metrics, falling back to the defaults for unset entries.
func (c LandscapeConfig) metricTiers() (high, second string) {
	high, second = defaultHighTier, defaultSecondTier
	if len(c.Tiers) > 0 && strings.TrimSpace(c.Tiers[0]) != "" {
		high = strings.TrimSpace(c.Tiers[0])
	}
	if len(c.Tiers) > 1 && strings.TrimSpace(c.Tiers[1]) != "" {
		second = strings.TrimSpace(c.Tiers[1])
	}
	return high, second
}
