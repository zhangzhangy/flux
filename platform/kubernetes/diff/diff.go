package diff

import ()

// Difference represents an individual difference between two
// `Object`s. This is an interface because
type Difference interface {
	String() string
}

type ObjectSetDiff struct {
	OnlyA     []Object
	OnlyB     []Object
	Different map[ObjectID][]Difference
}

// Diff calculates the differences between one model and another
func Diff(a, b ObjectSet) (ObjectSetDiff, error) {
	diff := ObjectSetDiff{}

	// A - B and A ^ B at the same time
	for id, objA := range a {
		if objB, found := b[id]; found {
			objDiff, err := DiffObject(objA, objB)
			if err != nil {
				return diff, err
			}
			diff.Different[id] = objDiff
		} else {
			diff.OnlyA = append(diff.OnlyA, objA)
		}
	}
	// now, B - A
	for id, objB := range b {
		if _, found := a[id]; !found {
			diff.OnlyB = append(diff.OnlyB, objB)
		}
	}
	return diff, nil
}
