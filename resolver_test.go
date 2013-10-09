package main

import (
	"github.com/aarondl/pack"
	. "testing"
)

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

func verifyDeps(deps map[string]*pack.Version, expdeps ...string) bool {
	if len(deps) != len(expdeps) {
		return false
	}

	for _, dep := range expdeps {
		d, err := pack.ParseDependency(dep)
		if err != nil {
			panic("Check input to this function carefully.")
		}
		if v, ok := deps[d.Name]; !ok ||
			!v.Satisfies(pack.Equal, d.Constraints[0].Version) {

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

	var err error
	var deps map[string]*pack.Version
	if deps, err = basic.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(basic.String())
	}

	expdeps := []string{`apple 1.0.0`, `banana 1.0.0`}
	if !verifyDeps(deps, expdeps...) {
		t.Error("Expected dependencies were not resolved:", expdeps)
		t.Error(deps)
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

	var err error
	var deps map[string]*pack.Version
	if deps, err = depthFirst.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(depthFirst.String())
	}

	expdeps := []string{`eggplant 1.0.0`, `banana 1.0.0`, `durian 1.0.0`}
	if !verifyDeps(deps, expdeps...) {
		t.Error("Expected dependencies were not resolved:", expdeps)
		t.Error(deps)
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

	var err error
	var deps map[string]*pack.Version
	if deps, err = constraints.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(constraints.String())
	}

	expdeps := []string{`apple 1.0.0`, `banana 1.0.0`}
	if !verifyDeps(deps, expdeps...) {
		t.Error("Expected dependencies were not resolved:", expdeps)
		t.Error(deps)
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

	var err error
	var deps map[string]*pack.Version
	if deps, err = backjump.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(backjump.String())
	}

	expdeps := []string{`apple 0.0.1`, `banana 0.0.1`, `durian 0.0.1`}
	if !verifyDeps(deps, expdeps...) {
		t.Error("Expected dependencies were not resolved:", expdeps)
		t.Error(deps)
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

	var err error
	var deps map[string]*pack.Version
	if deps, err = backjumpHeaven.solve(&repository); err != nil {
		t.Error("Solution was not found:", err)
		t.Error(backjumpHeaven.String())
	}

	expdeps := []string{
		`apple 0.0.1`, `banana 0.0.1`, `carrot 0.0.1`, `durian 0.0.1`,
	}
	if !verifyDeps(deps, expdeps...) {
		t.Error("Expected dependencies were not resolved:", expdeps)
		t.Error(deps)
	}

	if !verifySolution(backjumpHeaven) {
		t.Error("Solution could not be verified.")
		t.Error(backjumpHeaven.String())
	}
}
