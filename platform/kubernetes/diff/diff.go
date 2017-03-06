package diff

import (
	"errors"
)

// Difference represents an individual difference between two
// `Object`s. This is an interface because
type Difference interface {
	String() string
}

type ObjectSetDiff struct {
	OnlyA     []ObjectID
	OnlyB     []ObjectID
	Different map[ObjectID][]Difference
}

// Diff calculates the differences between one model and another
func Diff(a, b ObjectSet) (ObjectSetDiff, error) {
	return ObjectSetDiff{}, errors.New("not implemented")
}
