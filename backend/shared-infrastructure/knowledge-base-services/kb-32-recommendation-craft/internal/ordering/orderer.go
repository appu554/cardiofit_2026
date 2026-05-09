// Package ordering implements the canonical recommendation reordering stage
// of the six-stage rendering pipeline.
//
// VisibilityClass: PDP — recommendation reordering, never suppression
//
// Order applies a canonical type-based ordering so that the most actionable
// recommendation type (STOP) appears first. It uses sort.SliceStable to
// preserve the relative input order of packets that share the same type.
//
// Anti-suppression invariant: len(out) == len(in) always. The orderer's sole
// responsibility is sequencing; filtering is forbidden here. Any packet with
// an unrecognised Type is sorted to the end (treated as lowest priority).
package ordering

import (
	"math"
	"sort"

	"github.com/cardiofit/kb32/internal/generator"
)

// typeRank maps the canonical recommendation type strings to their sort priority.
// Lower values sort earlier. STOP is highest priority (0), ADD is lowest (3).
// Unrecognised types receive math.MaxInt via the rankOf helper and therefore
// sort to the very end of the output slice — they are never dropped.
var typeRank = map[string]int{
	"STOP":        0,
	"MONITOR":     1,
	"DOSE_CHANGE": 2,
	"ADD":         3,
}

// rankOf returns the sort rank for a packet type. Recognised types return their
// canonical rank; unrecognised types return math.MaxInt so they sort to the end.
// This is a deliberate design choice: unknown types are not an error — they pass
// through in order of appearance after all known types, preserving forward
// compatibility when new types are introduced.
func rankOf(packetType string) int {
	if r, ok := typeRank[packetType]; ok {
		return r
	}
	return math.MaxInt
}

// Order reorders packets by canonical recommendation type using a stable sort.
// The returned slice always has the same length as the input (anti-suppression
// invariant). Packets with unrecognised types appear after all known types,
// sorted stably among themselves.
//
// Order never returns nil: an empty input yields a non-nil empty slice.
func Order(in []*generator.Packet) []*generator.Packet {
	if len(in) == 0 {
		return []*generator.Packet{}
	}

	out := make([]*generator.Packet, len(in))
	copy(out, in)

	sort.SliceStable(out, func(i, j int) bool {
		return rankOf(out[i].Type) < rankOf(out[j].Type)
	})

	return out
}
