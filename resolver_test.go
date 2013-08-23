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
	graph *depgraph
}

func (tdgp *testDepGraphProvider) GetGraph() *depgraph {
	return tdgp.graph
}

var repository = testVersionProvider{map[string][]*pack.Version{
	"root":   mkVers("1.0.0"),
	"apple":  mkVers("1.0.0", "0.2.1", "0.0.1"),
	"banana": mkVers("1.0.0", "0.1.0", "0.1.0-dev", "0.0.1"),
	"carrot": mkVers("1.0.0", "0.1.0", "0.1.0-dev", "0.0.1"),
	"durian": mkVers("1.0.1", "1.0.0", "0.0.2", "0.0.1"),
}}

/*var basic = mkTree(`
root 1.0.0
-apple
-banana 0.0.1
`)*/

/*var backjumpHeaven = mkTree(`
root 1.0.0
-apple
--durian >=0.0.1
-banana
--durian <1.0.1
-carrot
--durian =0.0.1
`)*/

func verifySolution(d *depgraph) bool {
	for _, kid := range d.head.kids {
		if !verifySolutionHelper(kid) {
			return false
		}
	}

	return true
}

func verifySolutionHelper(d *depnode) bool {
	if d.v == nil {
		return false
	}
	for _, kid := range d.kids {
		if !verifySolutionHelper(kid) {
			return false
		}
	}

	return true
}

func TestSolver_Basic(t *T) {
	var basic = mkGraph(`
	root 1.0.0
	-apple
	-banana 0.0.1
	`)

	if !basic.solve(&repository) {
		t.Error("Solution was not found.")
	}

	if !verifySolution(basic) {
		t.Error("Solution could not be verified.")
	}
}

func TestSolver_DepthFirst(t *T) {
	var depthFirst = mkGraph(`
	root 1.0.0
	-apple
	--carrot
	-banana
	--durian
	`)

	t.Log(depthFirst.String())
	t.Log(depthFirst.head.d.String())
	t.Log(depthFirst.head.kids)
	t.Log(depthFirst.head.kids[0].d.String())
	t.Log(depthFirst.head.kids[0].kids)
	t.Log(depthFirst.head.kids[1].d.String())
	t.Log(depthFirst.head.kids[1].kids)

	if !depthFirst.solve(&repository) {
		t.Error("Solution was not found.")
	}

	if !verifySolution(depthFirst) {
		t.Error("Solution could not be verified.")
	}
}

func TestSolver_Constraints(t *T) {
	var constraints = mkGraph(`
	root 1.0.0
	-apple 0.0.1
	--carrot ~0.0.1
	-banana 0.0.1
	--durian 0.0.1
	`)

	if !constraints.solve(&repository) {
		t.Error("Solution was not found.")
	}

	if !verifySolution(constraints) {
		t.Error("Solution could not be verified.")
	}
}

func TestSolver_Backjumpheaven(t *T) {
	var backjumpHeaven = mkGraph(`
	root 1.0.0
	-apple
	--durian >=0.0.1
	-banana
	--durian <1.0.1
	-carrot
	--durian =0.0.1
	`)

	t.Log(backjumpHeaven.String())
	t.Log(backjumpHeaven.head.d.String())
	t.Log(backjumpHeaven.head.kids)
	t.Log(backjumpHeaven.head.kids[0].d.String())
	t.Log(backjumpHeaven.head.kids[0].kids)
	t.Log(backjumpHeaven.head.kids[1].d.String())
	t.Log(backjumpHeaven.head.kids[1].kids)
	t.Log(backjumpHeaven.head.kids[2].d.String())
	t.Log(backjumpHeaven.head.kids[2].kids)

	if !backjumpHeaven.solve(&repository) {
		t.Error("Solution was not found.")
	}

	if !verifySolution(backjumpHeaven) {
		t.Error("Solution could not be verified.")
	}
}
