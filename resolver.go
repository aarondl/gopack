package main

import (
	"bytes"
	"github.com/aarondl/pack"
)

const (
	rightpipe = '├'
	downpipe  = '┬'
	endpipe   = '└'
	horizpipe = '─'
	vertpipe  = '│'
	space     = ' '
	newline   = '\n'
)

// versionProvider allows us to look up available versions for each package.
type versionProvider interface {
	GetVersions(string) []*pack.Version
}

// depGraphProvider allows retrieval of a dependency graph to solve.
type depGraphProvider interface {
	GetGraph() *depGraph
}

// depNode is a dependency node.
type depNode struct {
	*pack.Dependency
	v    *pack.Version
	kids []*depNode
}

// depGraph is a dependency graph.
type depGraph struct {
	head *depNode
}

// solve solves a dependency graph.
func (d *depGraph) solve(graph *depGraph, vp versionProvider) bool {
	return false
}

// String turns a depgraph into a string.
func (d depGraph) String() string {
	var b bytes.Buffer

	depGraphVisualize(d.head, 0, 0, len(d.head.kids) == 0, &b)
	return b.String()
}

// depGraphVisualize builds a graph visualization.
func depGraphVisualize(n *depNode, depth, cur int, last bool, b *bytes.Buffer) {
	kids := len(n.kids)

	if depth > 0 {
		for i := 1; i < depth; i++ {
			if i < cur+1 {
				b.WriteRune(vertpipe)
			} else {
				b.WriteRune(space)
			}
			b.WriteRune(space)
		}
		if last {
			b.WriteRune(endpipe)
		} else {
			b.WriteRune(rightpipe)
		}
		if kids > 0 {
			b.WriteRune(horizpipe)
			b.WriteRune(downpipe)
		} else {
			b.WriteRune(horizpipe)
		}
		b.WriteRune(space)
	}

	b.WriteString(n.Dependency.String())
	if !last || kids > 0 || cur > 0 {
		b.WriteByte(newline)
	}

	for i := 0; i < kids; i++ {
		last = i+1 == kids
		curActive := cur
		if !last {
			curActive++
		}
		depGraphVisualize(n.kids[i], depth+1, curActive, last, b)
	}

	return
}
