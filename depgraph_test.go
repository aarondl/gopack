package main

import (
	"github.com/aarondl/pack"
	"strings"
	. "testing"
)

func mkVers(versions ...string) (v []*pack.Version) {
	ln := len(versions)
	if ln == 0 {
		panic("Why?")
	}

	var err error
	v = make([]*pack.Version, ln)
	for i := 0; i < ln; i++ {
		v[i], err = pack.ParseVersion(versions[i])
		if err != nil {
			panic("Write better versions:" + versions[i])
		}
	}
	return
}

func mkDep(dep string) (d *pack.Dependency) {
	var err error
	d, err = pack.ParseDependency(dep)
	if err != nil {
		panic("Write better dependencies:" + dep)
	}
	return
}

func mkGraph(graph string) (g *depgraph) {
	var err error
	g = new(depgraph)
	var stack = make([]*depnode, 0)
	var dep *pack.Dependency
	var previous, parent *depnode
	var stackindex = 0

	lines := strings.Split(graph, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if g.head == nil {
			if line[0] == '-' {
				panic("First line must be head.")
			}
			dep, err = pack.ParseDependency(line)
			if err != nil {
				panic("Bad dep:" + line)
			}
			previous = &depnode{d: dep, v: dep.Constraints[0].Version}
			g.head = previous
			parent = previous
			stack = append(stack, previous)
			stackindex = 0
		} else {
			depth := 0
			for line[0] == '-' {
				line = line[1:]
				depth++
			}

			curdepth := stackindex + 1

			if depth == 0 {
				panic("No head level elements allowed.")
			}
			if depth-curdepth > 1 {
				panic("Elements must have direct children.")
			}

			dep, err = pack.ParseDependency(line)
			if err != nil {
				panic("Bad dependency:" + line)
			}

			if depth > curdepth {
				parent = previous
				previous.kids = make([]*depnode, 0)
				stackindex++
				if stackindex >= len(stack) {
					stack = append(stack, previous)
				} else {
					stack[stackindex] = previous
				}
			} else if depth < curdepth {
				stackindex -= curdepth - depth
				parent = stack[stackindex]
			}

			previous = &depnode{d: dep}
			parent.kids = append(parent.kids, previous)
		}
	}

	return
}

var printTest = mkGraph(`
pack1 0.0.1
-pack3 ~1.0.0 !=1.1.2
--pack4 ~2.0.0
---pack5 ~3.0.0
----pack9 !=4.0.0
-----pack10 !=4.0.0
----pack11 !=4.0.0
--pack6 ~4.0.0
-pack7 >=5.0.0
--pack8 ~6.0.0
`)

func TestDepgraph_String(t *T) {
	expect :=
		"pack1 0.0.1 (=0.0.1)\n" +
			"├─┬ pack3 1.2.3 (~1.0.0 !=1.1.2)\n" +
			"│ ├─┬ pack4 (~2.0.0)\n" +
			"│ │ └─┬ pack5 (~3.0.0)\n" +
			"│ │   ├─┬ pack9 (!=4.0.0)\n" +
			"│ │   │ └─ pack10 (!=4.0.0)\n" +
			"│ │   └─ pack11 (!=4.0.0)\n" +
			"│ └─ pack6 (~4.0.0)\n" +
			"└─┬ pack7 (>=5.0.0)\n" +
			"  └─ pack8 (~6.0.0)"

	printTest.head.kids[0].v = &pack.Version{1, 2, 3, ``}
	if str := printTest.String(); str != expect {
		t.Errorf("Expected:\n%s\ngot:\n%s", expect, str)
	}
}
