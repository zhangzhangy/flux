package diff

import (
	"errors"
	"fmt"
	"reflect"
)

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

func MakeObjectSetDiff() ObjectSetDiff {
	return ObjectSetDiff{
		Different: map[ObjectID][]Difference{},
	}
}

// Diff calculates the differences between one model and another
func DiffSet(a, b ObjectSet) (ObjectSetDiff, error) {
	diff := MakeObjectSetDiff()

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

type Differ interface {
	Diff(a Differ, path string) ([]Difference, error)
}

// Diff one object with another. This assumes that the objects being
// compared are supposed to represent the same logical object, i.e.,
// they were identified with the same ID. An error indicates they are
// not comparable.
func DiffObject(a, b Object) ([]Difference, error) {
	if a.ID() != b.ID() {
		return nil, errors.New("objects being compared do not have the same ID")
	}

	// Special case at the top: if these have different runtime types,
	// they are not comparable.
	typA, typB := reflect.TypeOf(a), reflect.TypeOf(b)
	if typA != typB {
		return nil, errors.New("objects being compared are not the same runtime type")
	}
	return diffObj(reflect.ValueOf(a), reflect.ValueOf(b), typA, "")
}

var differType = reflect.TypeOf((*Differ)(nil)).Elem()

// Compare two values and compile a list of differences between them.
func diffObj(a, b reflect.Value, typ reflect.Type, path string) ([]Difference, error) {
	if typ.Implements(differType) {
		differA, differB := a.Interface().(Differ), b.Interface().(Differ)
		return differA.Diff(differB, path)
	}

	switch typ.Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		return diffArrayOrSlice(a, b, typ, path)
	case reflect.Interface:
		return nil, errors.New("interface diff not implemented")
	case reflect.Ptr:
		a, b, typ = reflect.Indirect(a), reflect.Indirect(b), typ.Elem()
		return diffObj(a, b, typ, path)
	case reflect.Struct:
		return diffStruct(a, b, typ, path)
	case reflect.Map:
		return nil, errors.New("map diff not implemented")
	case reflect.Func:
		return nil, errors.New("func dif not implemented (and not implementable)")
	default: // all ground types
		if a.Interface() != b.Interface() {
			return []Difference{makeValueDiff(a, b, path)}, nil
		}
		return nil, nil
	}
}

func makeValueDiff(a, b reflect.Value, path string) valueDifference {
	return valueDifference{a.Interface(), b.Interface(), path}
}

// A difference in ground values, e.g., in a field or in a slice element
type valueDifference struct {
	a, b interface{}
	path string
}

func (d valueDifference) String() string {
	return fmt.Sprintf("%v != %v", d.a, d.b)
}

// diff each exported field individually
func diffStruct(a, b reflect.Value, structTyp reflect.Type, path string) ([]Difference, error) {
	var diffs []Difference

	for i := 0; i < structTyp.NumField(); i++ {
		field := structTyp.Field(i)
		if field.PkgPath == "" { // i.e., is an exported field
			fieldDiffs, err := diffObj(a.Field(i), b.Field(i), field.Type, path+"."+field.Name)
			if err != nil {
				return nil, err
			}
			diffs = append(diffs, fieldDiffs...)
		}
	}
	return diffs, nil
}

// diff each element, report over- or underbite
func diffArrayOrSlice(a, b reflect.Value, sliceTyp reflect.Type, path string) ([]Difference, error) {
	diff := sliceDiff{
		Different: map[int][]Difference{},
	}
	elemTyp := sliceTyp.Elem()

	i := 0
	for ; i < a.Len() && i < b.Len(); i++ {
		d, err := diffObj(a.Index(i), b.Index(i), elemTyp, fmt.Sprintf("%s[%d]", path, i))
		if err != nil {
			return nil, err
		} else if len(d) > 0 {
			diff.Different[i] = d
		}
	}
	if i < a.Len()-1 {
		diff.OnlyA = sliceValue(a, i)
	} else if i < b.Len()-1 {
		diff.OnlyB = sliceValue(b, i)
	}
	if len(diff.OnlyA) > 0 || len(diff.OnlyB) > 0 || len(diff.Different) > 0 {
		return []Difference{diff}, nil
	}
	return nil, nil
}

func sliceValue(s reflect.Value, at int) []interface{} {
	res := make([]interface{}, s.Len()-at)
	for i := at; i < s.Len(); i++ {
		res[i-at] = s.Index(i).Interface()
	}
	return res
}

type sliceDiff struct {
	OnlyA     []interface{}
	OnlyB     []interface{}
	Different map[int][]Difference
}

func (s sliceDiff) String() string {
	return "slice diff"
}
