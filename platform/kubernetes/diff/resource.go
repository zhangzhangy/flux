package diff

import (
	"errors"
)

// https://kubernetes.io/docs/user-guide/identifiers/
// Objects are unique by {Namespace, Kind, Name}
type ObjectID struct {
	Namespace string
	Kind      string
	Name      string
}

type Object interface {
	ID() ObjectID
	// Source gives a string representation of the origin of the
	// Object definition, whether it's a file (possibly a part of a
	// file) or a cluster.
	Source() string
}

type baseObject struct {
	ObjectID
}

func (obj baseObject) ID() ObjectID {
	return obj.ObjectID
}

func (obj baseObject) Source() string {
	return "[base implementation]"
}

// Diff one object with another. This assumes that the objects being
// compared are supposed to represent the same logical object, i.e.,
// they were identified with the same ID.
func DiffObject(a, b Object) ([]Difference, error) {
	return nil, errors.New("not implemented")
}

// ObjectSet is a set of several objects which can be diffed
// collectively.
type ObjectSet map[ObjectID]Object
