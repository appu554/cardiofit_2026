package store

import (
	"database/sql/driver"
	"strings"
)

// pgStringArray is a minimal driver.Valuer for Postgres TEXT[] that avoids
// pulling lib/pq into the public surface of the store package. lib/pq is
// linked at the cmd/server entrypoint, where the *sql.DB is opened.
type pgStringArray []string

func (a pgStringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	// Build a Postgres array literal: {"a","b","c"}.
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

func ssToPGArray(ss []string) driver.Valuer { return pgStringArray(ss) }
