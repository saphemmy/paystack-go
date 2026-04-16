// Package testutil provides doubles and fixture helpers for the paystack-go
// test suite. It lives under internal/ so it cannot be imported outside this
// module.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// LoadFixture reads a file from testdata/ relative to the module root. It
// fails the test immediately if the fixture cannot be read.
func LoadFixture(t testing.TB, name string) []byte {
	t.Helper()
	path := filepath.Join(moduleRoot(), "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
		return nil
	}
	return data
}
