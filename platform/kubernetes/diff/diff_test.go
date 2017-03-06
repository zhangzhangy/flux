package diff

import (
	"reflect"
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

func TestSomeVsNone(t *testing.T) {
	objA := baseObject{
		ObjectID: ObjectID{
			Namespace: "a-namespace",
			Kind:      "Deployment",
			Name:      "a-name",
		},
	}

	setA := ObjectSet{
		objA.ObjectID: objA,
	}
	setB := ObjectSet{}

	diff, err := Diff(setA, setB)
	if err != nil {
		t.Error(err)
	}

	expected := ObjectSetDiff{
		OnlyA: []Object{objA},
	}
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

	setA := ObjectSet{}
	setB := ObjectSet{
		objB.ObjectID: objB,
	}

	diff, err := Diff(setA, setB)
	if err != nil {
		t.Error(err)
	}

	expected := ObjectSetDiff{
		OnlyB: []Object{objB},
	}
	if !reflect.DeepEqual(expected, diff) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, diff)
	}
}
