package main

import (
	"github.com/aarondl/pack"
	. "testing"
)

type testVersionProvider struct {
	versions map[string][]*pack.Version
}

func (tvp *testVersionProvider) GetVersions(name string) []*pack.Version {
	return tvp.versions[name]
}

type testDepGraphProvider struct {
	graph *depGraph
}

func (tdgp *testDepGraphProvider) GetGraph() *depGraph {
	return tdgp.graph
}

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

var repository = testVersionProvider{map[string][]*pack.Version{
	"pack1": mkVers("0.0.1", "0.2.1", "1.0.0"),
	"pack2": mkVers("0.0.1", "0.1.0-dev", "0.1.0", "1.0.0"),
}}

var basic = testDepGraphProvider{&depGraph{
	&depNode{
		mkDep("pack1 >=0.0.1"), nil, []*depNode{
			{mkDep("pack3 ~1.0.0"), nil, []*depNode{
				{mkDep("pack4 ~2.0.0"), nil, []*depNode{
					{mkDep("pack5 ~3.0.0"), nil, nil},
				}},
				{mkDep("pack6 ~4.0.0"), nil, nil},
			}},
			{mkDep("pack7 >=5.0.0"), nil, []*depNode{
				{mkDep("pack8 ~6.0.0"), nil, nil},
			}},
		},
	},
}}

func TestString(t *T) {
	expect :=
		"pack1 >=0.0.1\n" +
			"├─┬ pack3 ~1.0.0\n" +
			"│ ├─┬ pack4 ~2.0.0\n" +
			"│ │ └─ pack5 ~3.0.0\n" +
			"│ └─ pack6 ~4.0.0\n" +
			"└─┬ pack7 >=5.0.0\n" +
			"  └─ pack8 ~6.0.0"
	if str := basic.GetGraph().String(); str != expect {
		t.Errorf("Expected:\n%s\ngot:\n%s", expect, str)
	}
}
