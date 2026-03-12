package fhir

import (
	"io"
	"os"
	"strings"
)

// readFile reads the entire contents of a file.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// jsonReader creates an io.Reader from a JSON string.
func jsonReader(s string) io.Reader {
	return strings.NewReader(s)
}
