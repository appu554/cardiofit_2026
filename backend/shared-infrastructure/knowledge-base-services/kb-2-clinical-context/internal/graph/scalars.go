package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/99designs/gqlgen/graphql"
)

// DateTime scalar implementation
func MarshalDateTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		data, _ := json.Marshal(t.Format(time.RFC3339))
		w.Write(data)
	})
}

func UnmarshalDateTime(v interface{}) (time.Time, error) {
	str, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("DateTime must be a string")
	}
	return time.Parse(time.RFC3339, str)
}

// UUID scalar implementation
func MarshalUUID(id uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		data, _ := json.Marshal(id.String())
		w.Write(data)
	})
}

func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
	str, ok := v.(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("UUID must be a string")
	}
	return uuid.Parse(str)
}

// JSON scalar implementation
func MarshalJSON(data interface{}) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		jsonData, _ := json.Marshal(data)
		w.Write(jsonData)
	})
}

func UnmarshalJSON(v interface{}) (interface{}, error) {
	return v, nil
}