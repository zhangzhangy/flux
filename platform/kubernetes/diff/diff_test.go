package diff

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// --- test diffing objects

type TestValue struct {
	baseObject
	ignoreUnexported string
	StringField      string
	IntField         int
	Differ           TestDiffer
	DifferStar       *TestDiffer
	Embedded         struct {
		NestedValue bool
	}
}

type TestDiffer struct {
	CaseInsensitive string
}

func (t TestDiffer) Diff(d Differ, path string) ([]Difference, error) {
	var other *TestDiffer
	switch d := d.(type) {
	case TestDiffer:
		other = &d
	case *TestDiffer:
		other = d
	default:
		return nil, errors.New("not diffable values")
	}

	if !strings.EqualFold(t.CaseInsensitive, other.CaseInsensitive) {
		return []Difference{valueDifference{t.CaseInsensitive, other.CaseInsensitive, path}}, nil
	}
	return nil, nil
}

func TestFieldwiseDiff(t *testing.T) {
	id := ObjectID{
		Namespace: "namespace",
		Kind:      "TestFieldwise",
		Name:      "testcase",
	}
	a := TestValue{
		baseObject:       baseObject{id},
		ignoreUnexported: "one value",
		StringField:      "ground value",
		IntField:         5,
		Differ:           TestDiffer{"case-insensitive"},
		DifferStar:       &TestDiffer{"case-insensitive"},
	}
	a.Embedded.NestedValue = true

	b := TestValue{
		baseObject:       baseObject{id},
		ignoreUnexported: "completely different value",
		StringField:      "a different ground value",
		IntField:         7,
		Differ:           TestDiffer{"CASE-INSENSITIVE"},
		DifferStar:       &TestDiffer{"CASE-INSENSITIVE"},
	}
	b.Embedded.NestedValue = false

	diffs, err := DiffObject(a, a)
	if err != nil {
		t.Error(err)
	}
	if len(diffs) > 0 {
		t.Errorf("expected no diffs, got %#v", diffs)
	}

	diffs, err = DiffObject(a, b)
	if err != nil {
		t.Error(err)
	}
	if len(diffs) != 3 {
		t.Errorf("expected three diffs, got:\n%#v", diffs)
	}
}

// --- test whole `ObjectSet`s

func TestEmptyVsEmpty(t *testing.T) {
	setA := MakeObjectSet("A")
	setB := MakeObjectSet("B")
	diff, err := DiffSet(setA, setB)
	if err != nil {
		t.Error(err)
	}
	if len(diff.OnlyA) > 0 || len(diff.OnlyB) > 0 || len(diff.Different) > 0 {
		t.Errorf("expected no differences, got %#v", diff)
	}
}

func TestSomeVsNone(t *testing.T) {
	objA := baseObject{
		ObjectID: ObjectID{
			Namespace: "a-namespace",
			Kind:      "Deployment",
			Name:      "a-name",
		},
	}

	setA := MakeObjectSet("A")
	setA.Objects[objA.ObjectID] = objA
	setB := MakeObjectSet("B")

	diff, err := DiffSet(setA, setB)
	if err != nil {
		t.Error(err)
	}

	expected := MakeObjectSetDiff(setA, setB)
	expected.OnlyA = []Object{objA}
	if !reflect.DeepEqual(expected, diff) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, diff)
	}
}

func TestNoneVsSome(t *testing.T) {
	objB := baseObject{
		ObjectID: ObjectID{
			Namespace: "b-namespace",
			Kind:      "Deployment",
			Name:      "b-name",
		},
	}

	setA := MakeObjectSet("A")
	setB := MakeObjectSet("B")
	setB.Objects[objB.ObjectID] = objB

	diff, err := DiffSet(setA, setB)
	if err != nil {
		t.Error(err)
	}

	expected := MakeObjectSetDiff(setA, setB)
	expected.OnlyB = []Object{objB}
	if !reflect.DeepEqual(expected, diff) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, diff)
	}
}

func TestSliceDiff(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"a", "b'"}
	diffs, err := diffObj(reflect.ValueOf(a), reflect.ValueOf(b), reflect.TypeOf(a), "slice")
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected more than zero differences, but got zero")
	}

	expected := sliceDiff{
		path:  "slice",
		len:   2,
		OnlyA: []interface{}{"c"},
		Different: map[int][]Difference{
			1: []Difference{
				valueDifference{
					a:    "b",
					b:    "b'",
					path: "slice[1]",
				},
			},
		},
	}
	if !reflect.DeepEqual(expected, diffs[0]) {
		t.Errorf("expected one diff:\n%#v\ngot:\n%#v\n", expected, diffs[0])
	}
}
