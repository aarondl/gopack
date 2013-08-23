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

// depnode is a dependency node.
type depnode struct {
	d    *pack.Dependency
	v    *pack.Version
	kids []*depnode
}

// depgraph is a dependency graph.
type depgraph struct {
	head *depnode
}

// depgraphProvider allows retrieval of a dependency graph to solve.
type depgraphProvider interface {
	GetGraph() *depgraph
}

// String turns a depgraph into a string.
func (g depgraph) String() string {
	var b bytes.Buffer

	depgraphVisualize(g.head, 0, 0, len(g.head.kids) == 0, &b, true, true)
	return b.String()
}

// depgraphVisualize builds a graph visualization.
func depgraphVisualize(n *depnode, depth uint, active uint64, last bool,
	b *bytes.Buffer, showConstraints, showVersions bool) {

	kids := len(n.kids)

	if depth > 0 {
		var i uint
		for i = 1; i < depth; i++ {
			if active&(1<<(i-1)) != 0 {
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

	b.WriteString(n.d.Name)
	if showVersions && n.v != nil {
		b.WriteRune(space)
		b.WriteString(n.v.String())
	}
	if showConstraints && len(n.d.Constraints) > 0 {
		b.WriteRune(space)
		b.WriteByte('(')
		for i := 0; i < len(n.d.Constraints); i++ {
			if i != 0 {
				b.WriteRune(space)
			}
			b.WriteString(n.d.Constraints[i].Operator.String())
			b.WriteString(n.d.Constraints[i].Version.String())
		}
		b.WriteByte(')')
	}
	if !last || kids > 0 || active > 0 {
		b.WriteByte(newline)
	}

	for i := 0; i < kids; i++ {
		last = i+1 == kids
		tmpactive := active
		if !last {
			tmpactive |= 1 << depth
		}
		depgraphVisualize(n.kids[i], depth+1, tmpactive, last, b,
			showConstraints, showVersions)
	}

	return
}
