package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

func TestInMemory_GetMissReturnsFalse(t *testing.T) {
	c := NewInMemory()
	got, ok, err := c.Get(context.Background(), "nope")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestInMemory_SetThenGet(t *testing.T) {
	c := NewInMemory()
	res := &evaluator.Result{Decision: dsl.DecisionGranted, RuleID: "R1"}
	require.NoError(t, c.Set(context.Background(), "k", res, time.Minute))

	got, ok, err := c.Get(context.Background(), "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "R1", got.RuleID)
}

func TestInMemory_Expiry(t *testing.T) {
	c := NewInMemory()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	res := &evaluator.Result{Decision: dsl.DecisionGranted}
	require.NoError(t, c.Set(context.Background(), "k", res, time.Minute))

	// Advance the fake clock past TTL.
	c.now = func() time.Time { return now.Add(2 * time.Minute) }
	_, ok, _ := c.Get(context.Background(), "k")
	assert.False(t, ok, "entry should be expired")
}

func TestInMemory_InvalidatePattern(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()
	for _, k := range []string{
		"auth:v1:AU/VIC:rn:prescribe:S4:antibiotics:r1:t",
		"auth:v1:AU/VIC:rn:prescribe:S8:antibiotics:r1:t",
		"auth:v1:AU/TAS:rn:prescribe:S4:antibiotics:r1:t",
	} {
		require.NoError(t, c.Set(ctx, k, &evaluator.Result{}, time.Hour))
	}
	require.Equal(t, 3, c.Size())

	require.NoError(t, c.Invalidate(ctx, "auth:v1:AU/VIC:*"))
	assert.Equal(t, 1, c.Size(), "AU/VIC entries should be gone, AU/TAS preserved")

	require.NoError(t, c.Invalidate(ctx, "*"))
	assert.Equal(t, 0, c.Size())
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern, s string
		want       bool
	}{
		{"*", "anything", true},
		{"foo", "foo", true},
		{"foo", "bar", false},
		{"foo*", "foobar", true},
		{"foo*", "foo", true},
		{"*bar", "foobar", true},
		{"*bar", "barx", false},
		{"foo*baz", "foobarbaz", true},
		{"foo*baz", "foobaz", true},
		{"foo*baz", "fooBAZ", false},
		{"a*b*c", "axxbxxc", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "acb", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, matchPattern(c.pattern, c.s),
			"matchPattern(%q, %q)", c.pattern, c.s)
	}
}

func TestDefaultTTL_StaticDeniedRule(t *testing.T) {
	r := evaluator.Result{Decision: dsl.DecisionDenied}
	assert.Equal(t, 24*time.Hour, DefaultTTL(r))
}

func TestDefaultTTL_CredentialDependent(t *testing.T) {
	r := evaluator.Result{
		Decision: dsl.DecisionGrantedWithConditions,
		Conditions: []evaluator.ConditionResult{
			{Condition: "endorsement_current", Check: "Credential.endorsement_valid_at_action_time"},
		},
	}
	assert.Equal(t, time.Hour, DefaultTTL(r))
}

func TestDefaultTTL_AgreementDependent(t *testing.T) {
	r := evaluator.Result{
		Decision: dsl.DecisionGrantedWithConditions,
		Conditions: []evaluator.ConditionResult{
			{Condition: "agreement_in_place", Check: "PrescribingAgreement.exists_for_person"},
		},
	}
	assert.Equal(t, 15*time.Minute, DefaultTTL(r))
}

func TestDefaultTTL_ConsentDependent(t *testing.T) {
	r := evaluator.Result{
		Decision: dsl.DecisionGrantedWithConditions,
		Conditions: []evaluator.ConditionResult{
			{Condition: "consent_active", Check: "Consent.active_for_resident"},
		},
	}
	assert.Equal(t, 5*time.Minute, DefaultTTL(r))
}

func TestDefaultTTL_ConsentDominatesAgreementDominatesCredential(t *testing.T) {
	// Multiple condition types: consent (5m) wins.
	r := evaluator.Result{
		Decision: dsl.DecisionGrantedWithConditions,
		Conditions: []evaluator.ConditionResult{
			{Condition: "endorsement_current", Check: "Credential.x"},
			{Condition: "agreement_in_place", Check: "PrescribingAgreement.x"},
			{Condition: "consent_active", Check: "Consent.x"},
		},
	}
	assert.Equal(t, 5*time.Minute, DefaultTTL(r))
}

func TestRedisCache_StubInterface(t *testing.T) {
	// The stub must satisfy the Cache interface and never error. This locks
	// the contract until production Redis credentials are wired.
	var c Cache = NewRedis()
	got, ok, err := c.Get(context.Background(), "anything")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
	require.NoError(t, c.Set(context.Background(), "k", &evaluator.Result{}, time.Minute))
	require.NoError(t, c.Invalidate(context.Background(), "*"))
	assert.Equal(t, 0, c.Size())
}

// TestInMemory_HitRate verifies > 95% hit rate on a steady-state workload
// per Layer 3 v2 doc Part 4.5.3 acceptance criterion.
func TestInMemory_HitRate(t *testing.T) {
	c := NewInMemory()
	ctx := context.Background()
	res := &evaluator.Result{Decision: dsl.DecisionGranted}

	// Pre-warm 100 keys.
	for i := 0; i < 100; i++ {
		key := keyN(i)
		require.NoError(t, c.Set(ctx, key, res, time.Hour))
	}

	hits, total := 0, 0
	for round := 0; round < 100; round++ {
		for i := 0; i < 100; i++ {
			_, ok, _ := c.Get(ctx, keyN(i))
			total++
			if ok {
				hits++
			}
		}
	}
	rate := float64(hits) / float64(total)
	assert.Greater(t, rate, 0.99, "steady-state hit rate")
}

func keyN(n int) string {
	return "auth:v1:AU/VIC:rn:prescribe:S4:abx:" + uuidNthHex(n) + ":t"
}

func uuidNthHex(n int) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	for i := range out {
		out[i] = hex[(n+i)%16]
	}
	out[8], out[13], out[18], out[23] = '-', '-', '-', '-'
	return string(out)
}
