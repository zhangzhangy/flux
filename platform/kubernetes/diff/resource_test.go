package diff

import (
	"testing"
)

var serviceA = `---
kind: Service
meta:
  name: ServiceA
  namespace: NamespaceA
spec:
  type: NodePort
  ports:
    - port: 80
  selector:
    name: appA
`

var serviceB = `---
kind: Service
meta:
  name: ServiceA
  namespace: NamespaceA
spec:
  type: LoadBalancer
  ports:
    - port: 443
  selector:
    label: foo
    name: appB
`

func TestServiceDiff(t *testing.T) {
	a, err := ParseMultidoc([]byte(serviceA))
	if err != nil {
		t.Fatal(err)
	}

	b, err := ParseMultidoc([]byte(serviceB))
	if err != nil {
		t.Fatal(err)
	}

	diff, err := DiffSet(a, b)
	if err != nil {
		t.Error(err)
	}

	if len(diff.OnlyA) > 0 {
		t.Errorf("expected no items just in A, got:\n%#v", diff.OnlyA)
	}
	if len(diff.OnlyB) > 0 {
		t.Errorf("expected no items just in B, got:\n%#v", diff.OnlyB)
	}
	if len(diff.Different) != 0 {
		t.Errorf("expected A and B different, got:\n%#v", diff.Different)
	}
}
