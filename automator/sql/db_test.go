package sql

import (
	"testing"
)

func TestDBCreate(t *testing.T) {
	if _, err := New("ql-mem", "automation"); err != nil {
		t.Fatal(err)
	}
}
