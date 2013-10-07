package main

import (
	"github.com/aarondl/pack"
	"log"
	"os"
	. "testing"
)

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	return
	if err == nil {
		log.SetOutput(f)
	}
}

type testvp struct {
	graphs map[string][]*depgraph
}

func (tvp *testvp) GetVersions(name string) (vs []*pack.Version) {
	if graphs, ok := tvp.graphs[name]; ok {
		vs = make([]*pack.Version, len(graphs))
		for i := 0; i < len(graphs); i++ {
			vs[i] = graphs[i].head.v
		}
	}
	return
}

func (tvp *testvp) GetGraph(name string, version *pack.Version) *depgraph {
	if graphs, ok := tvp.graphs[name]; ok {
		for i := 0; i < len(graphs); i++ {
			if graphs[i].head.v.Satisfies(pack.Equal, version) {
				return graphs[i]
			}
		}
	}
	return nil
}

type testDepGraphProvider struct {
	graph *depgraph
}

func (tdgp *testDepGraphProvider) GetGraph() *depgraph {
	return tdgp.graph
}

var repository = testvp{map[string][]*depgraph{
	`apple`: []*depgraph{
		mkGraph(
			`apple 1.0.0`,
		),
		mkGraph(
			`apple 0.0.1
			-durian >=0.0.1`,
		),
	},
	`banana`: []*depgraph{
		mkGraph(
			`banana 1.0.0`,
		),
		mkGraph(
			`banana 0.0.1
			-durian <=0.0.5`,
		),
	},
	`carrot`: []*depgraph{
		mkGraph(
			`carrot 1.0.0`,
		),
		mkGraph(
			`carrot 0.0.1
			-durian =0.0.1`,
		),
	},
	`durian`: []*depgraph{
		mkGraph(`
			durian 1.0.0
		`),
		mkGraph(`
			durian 0.0.5
		`),
		mkGraph(`
			durian 0.0.1
		`),
	},
	`eggplant`: []*depgraph{
		mkGraph(`
			eggplant 1.0.0
			-durian =1.0.0
		`),
		mkGraph(`
			eggplant 0.0.1
		`),
	},
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
	-banana
	`)

	if _, err := basic.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(basic.String())
	}

	if !verifySolution(basic) {
		t.Error("Solution could not be verified.")
		t.Error(basic.String())
	}
}

func TestSolver_DepthFirst(t *T) {
	var depthFirst = mkGraph(`
	root 1.0.0
	-eggplant
	-banana
	`)

	if _, err := depthFirst.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(depthFirst.String())
	}

	if !verifySolution(depthFirst) {
		t.Error("Solution could not be verified.")
		t.Error(depthFirst.String())
	}
}

func TestSolver_Constraints(t *T) {
	var constraints = mkGraph(`
	root 1.0.0
	-apple =1.0.0
	-banana >=0.0.2
	`)

	if _, err := constraints.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(constraints.String())
	}

	if !verifySolution(constraints) {
		t.Error("Solution could not be verified.")
		t.Error(constraints.String())
	}
}

func TestSolver_Backjump(t *T) {
	var backjump = mkGraph(`
	root 1.0.0
	-apple 0.0.1
	-banana 0.0.1
	`)

	if _, err := backjump.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(backjump.String())
	}

	if !verifySolution(backjump) {
		t.Error("Solution could not be verified.")
		t.Error(backjump.String())
	}
}

func TestSolver_Backjumpheaven(t *T) {
	var backjumpHeaven = mkGraph(`
	root 1.0.0
	-apple 0.0.1
	-banana 0.0.1
	-carrot 0.0.1
	`)

	if _, err := backjumpHeaven.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(backjumpHeaven.String())
	}

	if !verifySolution(backjumpHeaven) {
		t.Error("Solution could not be verified.")
		t.Error(backjumpHeaven.String())
	}
}
