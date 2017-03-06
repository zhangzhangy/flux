package diff

import (
	"testing"
)

func TestEmptyVsEmpty(t *testing.T) {
	setA := ObjectSet{}
	setB := ObjectSet{}
	diff, err := Diff(setA, setB)
	if err != nil {
		t.Error(err)
	}
	if len(diff.OnlyA) > 0 || len(diff.OnlyB) > 0 || len(diff.Different) > 0 {
		t.Errorf("expected no differences, got %#v", diff)
	}
}
