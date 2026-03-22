package enrollment

// ChannelVariant encapsulates channel-specific enrollment behaviour.
// Corporate, insurance, and government pathways differ in consent
// requirements, slot overrides, and review queue routing.
type ChannelVariant struct {
	ChannelType     ChannelType
	RequiresABHA    bool     // Government channel mandates ABHA linking
	RequiresConsent []string // Additional consent documents required
	SkipSlots       []string // Slots that can be auto-populated from channel data
	ReviewRouting   string   // Default risk stratum for review queue (HIGH, MEDIUM, LOW)
}

// channelVariants defines per-channel customizations.
var channelVariants = map[ChannelType]ChannelVariant{
	ChannelCorporate: {
		ChannelType:     ChannelCorporate,
		RequiresABHA:    false,
		RequiresConsent: []string{"corporate_health_program", "data_sharing"},
		SkipSlots:       []string{"ethnicity", "primary_language"},
		ReviewRouting:   "MEDIUM",
	},
	ChannelInsurance: {
		ChannelType:     ChannelInsurance,
		RequiresABHA:    false,
		RequiresConsent: []string{"insurance_wellness", "data_sharing", "claims_authorization"},
		SkipSlots:       []string{},
		ReviewRouting:   "MEDIUM",
	},
	ChannelGovernment: {
		ChannelType:     ChannelGovernment,
		RequiresABHA:    true, // ABDM mandate for government channel
		RequiresConsent: []string{"abdm_consent", "dpdpa_consent", "data_sharing"},
		SkipSlots:       []string{},
		ReviewRouting:   "HIGH", // Government patients often higher acuity
	},
}

// GetVariant returns the ChannelVariant for the given channel type.
// Returns the corporate variant as default if the channel is unknown.
func GetVariant(ct ChannelType) ChannelVariant {
	if v, ok := channelVariants[ct]; ok {
		return v
	}
	return channelVariants[ChannelCorporate]
}

// ShouldSkipSlot returns true if the given slot can be auto-populated
// (and therefore skipped during interactive collection) for this channel.
func (v ChannelVariant) ShouldSkipSlot(slotName string) bool {
	for _, s := range v.SkipSlots {
		if s == slotName {
			return true
		}
	}
	return false
}

// PendingConsents returns the list of consent documents that must be
// collected from the patient before enrollment can proceed.
func (v ChannelVariant) PendingConsents(alreadyCollected []string) []string {
	collected := make(map[string]bool, len(alreadyCollected))
	for _, c := range alreadyCollected {
		collected[c] = true
	}
	var pending []string
	for _, req := range v.RequiresConsent {
		if !collected[req] {
			pending = append(pending, req)
		}
	}
	return pending
}
