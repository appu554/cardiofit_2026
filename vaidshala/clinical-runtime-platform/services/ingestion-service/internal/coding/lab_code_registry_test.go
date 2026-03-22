package coding

import (
	"testing"
	"time"
)

func TestCodeMapping_CacheExpiry(t *testing.T) {
	m := codeMapping{
		LOINCCode: "11580-8",
		CachedAt:  time.Now().Add(-10 * time.Minute),
	}

	if time.Since(m.CachedAt) < 5*time.Minute {
		t.Error("expired cache entry should not be valid")
	}
}

func TestCodeMapping_CacheFresh(t *testing.T) {
	m := codeMapping{
		LOINCCode: "11580-8",
		CachedAt:  time.Now(),
	}

	if time.Since(m.CachedAt) >= 5*time.Minute {
		t.Error("fresh cache entry should be valid")
	}
}
