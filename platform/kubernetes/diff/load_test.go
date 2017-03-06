package diff

import (
	"reflect"
	"testing"

	"github.com/weaveworks/flux/platform/kubernetes/testdata"
)

func TestParseEmpty(t *testing.T) {
	doc := ``

	objs, err := ParseMultidoc([]byte(doc))
	if err != nil {
		t.Error(err)
	}
	if len(objs) != 0 {
		t.Errorf("expected empty set; got %#v", objs)
	}
}

func TestParseSome(t *testing.T) {
	docs := `---
kind: Service
metadata:
  name: b-service
  namespace: b-namespace
---
kind: Deployment
metadata:
  name: a-deployment
`
	objs, err := ParseMultidoc([]byte(docs))
	if err != nil {
		t.Error(err)
	}

	idA := ObjectID{"default", "Deployment", "a-deployment"}
	idB := ObjectID{"b-namespace", "Service", "b-service"}
	expected := ObjectSet{
		idA: baseObject{ObjectID: idA},
		idB: baseObject{ObjectID: idB},
	}

	if !reflect.DeepEqual(expected, objs) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, objs)
	}
}

func TestLoadSome(t *testing.T) {
	dir, cleanup := testdata.TempDir(t)
	defer cleanup()
	if err := testdata.WriteTestFiles(dir); err != nil {
		t.Fatal(err)
	}
	objs, err := Load(dir)
	if err != nil {
		t.Error(err)
	}
	// assume it's one per file for the minute
	if len(objs) != len(testdata.Files) {
		t.Errorf("expected %d objects from %d files, got result:\n%#v", len(testdata.Files), len(testdata.Files), objs)
	}
}
