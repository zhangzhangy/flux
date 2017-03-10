package diff

import (
	"reflect"
	"testing"

	"github.com/weaveworks/flux/diff"
	"github.com/weaveworks/flux/platform/kubernetes/testdata"
)

// for convenience
func base(kind, namespace, name string) baseObject {
	b := baseObject{Kind: kind}
	b.Meta.Namespace = namespace
	b.Meta.Name = name
	return b
}

func TestParseEmpty(t *testing.T) {
	doc := ``

	objs, err := ParseMultidoc([]byte(doc), "test")
	if err != nil {
		t.Error(err)
	}
	if len(objs.Objects) != 0 {
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
	objs, err := ParseMultidoc([]byte(docs), "test")
	if err != nil {
		t.Error(err)
	}

	objA := base("Deployment", "default", "a-deployment")
	objB := base("Service", "b-namespace", "b-service")
	expected := diff.MakeObjectSet("test")
	expected.Objects = map[diff.ObjectID]diff.Object{
		objA.ID(): &Deployment{baseObject: objA},
		objB.ID(): &Service{baseObject: objB},
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
	if len(objs.Objects) != len(testdata.Files) {
		t.Errorf("expected %d objects from %d files, got result:\n%#v", len(testdata.Files), len(testdata.Files), objs)
	}
}
