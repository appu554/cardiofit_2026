package invalidation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

func seedCache(t *testing.T, c cache.Cache, keys ...string) {
	t.Helper()
	for _, k := range keys {
		require.NoError(t, c.Set(context.Background(), k,
			&evaluator.Result{Decision: dsl.DecisionGranted}, time.Hour))
	}
}

func TestInvalidate_CredentialExpired_AffectsRoleScopedKeys(t *testing.T) {
	c := cache.NewInMemory()
	seedCache(t, c,
		"auth:v1:AU/VIC:designated_rn_prescriber:prescribe:S4:abx:r1:t",
		"auth:v1:AU:designated_rn_prescriber:prescribe:S4:abx:r2:t",
		"auth:v1:AU/VIC:registered_nurse:observe:::r1:t",
	)
	require.Equal(t, 3, c.Size())

	inv := New(c)
	require.NoError(t, inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{
		Type: EventCredentialExpired,
		Role: "designated_rn_prescriber",
	}))

	assert.Equal(t, 1, c.Size(), "only registered_nurse entry should remain")
}

func TestInvalidate_PrescribingAgreement_AffectsResidentKeys(t *testing.T) {
	c := cache.NewInMemory()
	residentA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	residentB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	seedCache(t, c,
		"auth:v1:AU/VIC:rn:prescribe:S4:abx:"+residentA.String()+":t",
		"auth:v1:AU/VIC:rn:prescribe:S8:opioids:"+residentA.String()+":t",
		"auth:v1:AU/VIC:rn:prescribe:S4:abx:"+residentB.String()+":t",
	)

	inv := New(c)
	require.NoError(t, inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{
		Type:        EventPrescribingAgreementChange,
		ResidentRef: &residentA,
	}))

	assert.Equal(t, 1, c.Size(), "only resident B entry should remain")
}

func TestInvalidate_ScopeRuleDeployed_AffectsJurisdiction(t *testing.T) {
	c := cache.NewInMemory()
	seedCache(t, c,
		"auth:v1:AU/VIC:rn:prescribe:S4:abx:r1:t",
		"auth:v1:AU/TAS:rn:prescribe:S4:abx:r1:t",
		"auth:v1:AU:rn:prescribe:S4:abx:r1:t",
	)
	inv := New(c)
	require.NoError(t, inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{
		Type:         EventScopeRuleDeployed,
		Jurisdiction: "AU/VIC",
	}))
	assert.Equal(t, 2, c.Size())
}

func TestInvalidate_ConsentChanged_AffectsResident(t *testing.T) {
	c := cache.NewInMemory()
	resident := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	other := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	seedCache(t, c,
		"auth:v1:AU/VIC:rn:prescribe:S4:abx:"+resident.String()+":t",
		"auth:v1:AU/VIC:rn:prescribe:S4:abx:"+other.String()+":t",
	)
	inv := New(c)
	require.NoError(t, inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{
		Type:        EventConsentChanged,
		ResidentRef: &resident,
	}))
	assert.Equal(t, 1, c.Size())
}

// TestInvalidate_LatencyUnderOneSecond verifies the 1s acceptance criterion
// from the plan (Wave 4B Task 5).
func TestInvalidate_LatencyUnderOneSecond(t *testing.T) {
	c := cache.NewInMemory()
	for i := 0; i < 1000; i++ {
		seedCache(t, c, "auth:v1:AU/VIC:rn:prescribe:S4:abx:resident-"+itoa(i)+":t")
	}
	inv := New(c)
	start := time.Now()
	require.NoError(t, inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{
		Type:         EventScopeRuleDeployed,
		Jurisdiction: "AU/VIC",
	}))
	elapsed := time.Since(start)
	assert.Less(t, elapsed, time.Second, "1000-key invalidation must complete in <1s")
	assert.Equal(t, 0, c.Size())
}

func TestInvalidator_NilCacheReturnsError(t *testing.T) {
	inv := New(nil)
	err := inv.InvalidateOnEvent(context.Background(), SubstrateChangeEvent{Type: EventConsentChanged})
	require.Error(t, err)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
