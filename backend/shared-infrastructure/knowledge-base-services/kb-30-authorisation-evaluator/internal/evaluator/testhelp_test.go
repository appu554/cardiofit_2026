package evaluator

import (
	"os"
	"path/filepath"
)

// readExample is a test helper that locates examples/ by walking upward.
func readExample(name string) ([]byte, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "examples", name)
		if data, err := os.ReadFile(candidate); err == nil {
			return data, nil
		}
		dir = filepath.Dir(dir)
	}
	return nil, os.ErrNotExist
}
