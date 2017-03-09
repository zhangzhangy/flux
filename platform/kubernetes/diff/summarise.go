package diff

import (
	"fmt"
	"io"
)

func (d changed) Summarise(out io.Writer) {
	fmt.Fprintf(out, "* %s: %v != %v\n", d.path, d.a, d.b)
}

func (d added) Summarise(out io.Writer) {
	fmt.Fprintf(out, "+ %s: %+v\n", d.path, d.value)
}

func (d removed) Summarise(out io.Writer) {
	fmt.Fprintf(out, "- %s: %+v\n", d.path, d.value)
}

func (d opaqueChanged) Summarise(out io.Writer) {
	fmt.Fprintf(out, "* %s: data has changed\n", d.path)
}

func (d ObjectSetDiff) Summarise(out io.Writer) {
	if len(d.OnlyA) > 0 {
		fmt.Fprintf(out, "Only in %s:\n", d.A.Source)
		for _, obj := range d.OnlyA {
			id := obj.ID()
			fmt.Fprintf(out, "%s %s/%s\n", id.Kind, id.Namespace, id.Name)
		}
	}
	if len(d.OnlyB) > 0 {
		fmt.Fprintf(out, "Only in %s:\n", d.B.Source)
		for _, obj := range d.OnlyB {
			id := obj.ID()
			fmt.Fprintf(out, "%s %s/%s\n", id.Kind, id.Namespace, id.Name)
		}
	}
	if len(d.Different) > 0 {
		for id, diffs := range d.Different {
			fmt.Fprintf(out, "%s %s/%s is different:\n", id.Kind, id.Namespace, id.Name)
			for _, diff := range diffs {
				diff.Summarise(out)
			}
		}
	}
}
