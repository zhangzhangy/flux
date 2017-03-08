package diff

import (
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Difference represents an individual difference between two
// `Object`s. This is an interface because
type Difference interface {
	Summarise(out io.Writer)
}

type ObjectSetDiff struct {
	A, B      *ObjectSet
	OnlyA     []Object
	OnlyB     []Object
	Different map[ObjectID][]Difference
}

func MakeObjectSetDiff(a, b *ObjectSet) ObjectSetDiff {
	return ObjectSetDiff{
		A:         a,
		B:         b,
		Different: map[ObjectID][]Difference{},
	}
}

// Diff calculates the differences between one model and another
func DiffSet(a, b *ObjectSet) (ObjectSetDiff, error) {
	diff := MakeObjectSetDiff(a, b)

	// A - B and A ^ B at the same time
	for id, objA := range a.Objects {
		if objB, found := b.Objects[id]; found {
			objDiff, err := DiffObject(objA, objB)
			if err != nil {
				return diff, err
			}
			if len(objDiff) > 0 {
				diff.Different[id] = objDiff
			}
		} else {
			diff.OnlyA = append(diff.OnlyA, objA)
		}
	}
	// now, B - A
	for id, objB := range b.Objects {
		if _, found := a.Objects[id]; !found {
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

func (d valueDifference) Summarise(out io.Writer) {
	fmt.Fprintf(out, "%s: %v != %v\n", d.path, d.a, d.b)
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
		path:      path,
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
	diff.len = i

	if i < a.Len() {
		diff.OnlyA = sliceValue(a, i)
	} else if i < b.Len() {
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
	path string
	len  int

	OnlyA     []interface{}
	OnlyB     []interface{}
	Different map[int][]Difference
}

func (s sliceDiff) Summarise(out io.Writer) {
	for _, diffs := range s.Different {
		for _, diff := range diffs {
			diff.Summarise(out)
		}
	}
	for i, removed := range s.OnlyA {
		fmt.Fprintf(out, "Removed %s[%d]: %+v\n", s.path, s.len+i, removed)
	}
	for i, added := range s.OnlyB {
		fmt.Fprintf(out, "Added %s[%d]: %+v\n", s.path, s.len+i, added)
	}
}
