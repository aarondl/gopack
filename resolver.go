package main

import (
	"bytes"
	"fmt"
	"github.com/aarondl/pack"
	"log"
)

const (
	initialStackSize = 20
)

// versionProvider allows us to look up available versions for each package
// the array returned must be in reverse sorted order for the best result from
// the solver as it assumes [0] > [1] > [2]...
type versionProvider interface {
	// GetVersions gets the versions only from the dependency information.
	GetVersions(string) []*pack.Version
	// GetGraphs gets a list of graphs showing dependency information for each
	// version of the package.
	GetGraph(string, *pack.Version) *depgraph
}

// stacknode helps to emulate recursion and perform safejumps.
type stacknode struct {
	kid     int
	vi      int
	ai      int
	current *depnode
	parent  *depnode
}

// savestate is the state of the algorithm at an activation point.
type savestate struct {
	*stacknode
	stack []stacknode
}

// activation is the details of a packages activation.
type activation struct {
	*pack.Dependency
	version *pack.Version
	state   *savestate
}

// String is used to debug activations.
func (a activation) String() string {
	var buf bytes.Buffer
	buf.WriteString(a.Name)
	if a.version != nil {
		buf.WriteRune(space)
		buf.WriteString(a.version.String())
	}
	return buf.String()
}

/*
solve a dependency graph. This algorithm is a depth first search with
backjumping to resolve conflicts.
*/
func (g *depgraph) solve(vp versionProvider) (map[string]*activation, error) {
	if len(g.head.kids) == 0 {
		return nil, nil
	}

	var current, parent *depnode = g.head, nil
	var stack = make([]stacknode, 0, initialStackSize)
	var ai, kid = -1, 0
	var activations []*activation
	var active *activation
	var versions = make(map[string][]*pack.Version)
	var version *pack.Version
	var vs []*pack.Version
	var vi int
	var ok bool
	var conflicts = make([]string, 0)
	var conflict bool

	var verbose = *DEBUG

	// setState is used to climb the stack, or restore a savestate
	var setState = func(sn *stacknode) {
		kid = sn.kid
		vi = sn.vi
		ai = sn.ai
		current = sn.current
		parent = sn.parent
	}

	for i := 0; i < 100; i++ {
		name := current.d.Name
		if verbose {
			log.Println("Current:", current.d)
		}

		// Don't process head.
		if current == g.head {
			if kid >= len(current.kids) {
				if verbose {
					log.Println("Success!")
				}
				break
			} else {
				goto NEXT
			}
		}

		// Skip Activation if we're on any child other than 0.
		if kid != 0 {
			goto NEXT
		}

		// Fetch Versions for current.
		vs = nil
		if vs, ok = versions[name]; !ok {
			vstmp := vp.GetVersions(current.d.Name)
			vs = make([]*pack.Version, len(vstmp))
			copy(vs, vstmp)
			versions[name] = vs
			if verbose {
				log.Printf("Fetched Versions")
			}
		}

		if verbose {
			log.Println("Versions:", vs)
		}

		// Check for activeness. The first activation will always serve as the
		// main activation point, with the others simply being save points.
		active = nil
		for j := 0; j < len(activations); j++ {
			if activations[j].Name == name {
				active = activations[j]
				break
			}
		}

		version = nil
		conflict = false
		if active != nil {
			if verbose {
				log.Println("Found activation:", active)
			}

			// Check that we comply with the currently active.
			for _, con := range current.d.Constraints {
				if !active.version.Satisfies(con.Operator, con.Version) {
					// We've found a problem.
					if verbose {
						log.Printf("Conflict: %v %v fails constraint: %v%v",
							name, active.version, con.Operator, con.Version)
					}

					conflict = true
					break
				}
			}

			version = active.version
		} else {
			if verbose {
				log.Println("Not activated:", name)
			}
			// Find a suitable version
			for ; version == nil && vi < len(vs); vi++ {
				if len(current.d.Constraints) == 0 {
					version = vs[vi]
					break
				}
				for _, con := range current.d.Constraints {
					if vs[vi].Satisfies(con.Operator, con.Version) {
						version = vs[vi]
						break
					}
				}
			}

			if version == nil {
				if verbose {
					log.Printf("Conflict: %v has no usable versions %v",
						name, vs)
				}
				conflict = true
			}
		}

		if conflict {
			conflicts = append(conflicts, name)
			// If we cannot climb the stack any further, go back to a save
			// point if one exists.
			if parent == g.head {
				if len(conflicts) == 0 {
					// No conflicts exist to jump back to.
					return nil, fmt.Errorf("We've tried everything mate: %v",
						conflicts)
				}
				name = conflicts[0]
				conflicts = conflicts[1:]
				var st *savestate
				for i := 0; i < len(activations); i++ {
					if activations[i].Name == name {
						st = activations[i].state
						break
					}
				}
				if st == nil {
					return nil,
						fmt.Errorf("Conflict's activation not found: %v %v",
							name, activations,
						)
				}
				setState(st.stacknode)
				vi++
				kid = 0
				stack = st.stack
				activations = activations[:ai]
				if verbose {
					log.Println("Conflict! Restoring:", current.d.Name)
					log.Println("Activations:", activations)
				}
				continue
			}

			// We can still climb the stack, try it.
			setState(&stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			vi++
			kid = 0
			activations = activations[:ai]
			if verbose {
				log.Println("Conflict! Popping:", current.d.Name)
				log.Println("Activations:", activations)
			}
			continue
		}

		// Add ourselves to the list of activators.
		ai++
		activations = append(activations,
			&activation{current.d, version, &savestate{
				&stacknode{kid, vi, ai, current, parent},
				make([]stacknode, len(stack)),
			}},
		)
		copy(activations[len(activations)-1].state.stack, stack)
		current.v = version

		if verbose {
			log.Printf("Added: %v %v to activations", name, version)
			log.Println("Activations:", activations)
			log.Println("Fetching Dependencies for:", name)
		}

		current.kids = vp.GetGraph(name, version).head.kids
		if verbose {
			var b bytes.Buffer
			for _, dep := range current.kids {
				b.WriteString(dep.d.String())
				b.WriteRune(space)
			}
			log.Println("Got dependencies:", b.String())
		}

	NEXT:
		// Push current on to stack, go into child.
		if kid < len(current.kids) {
			if verbose {
				log.Println("Pushing:", name, kid)
			}
			stack = append(stack, stacknode{kid, vi, ai, current, parent})
			parent, current = current, current.kids[kid]
			ai = len(activations) - 1
			kid, vi = 0, 0
			continue
		}

		// Pop off the stack back to parent.
		setState(&stack[len(stack)-1])
		if verbose {
			log.Println("Popping:", current.d.Name)
		}
		stack = stack[:len(stack)-1]
		kid++
	}

	// Remove the duplicates from the list of activations, check that only
	// a single version has been activated for sanity.
	dedupActs := make(map[string]*activation)
	for _, a := range activations {
		if active, ok = dedupActs[a.Name]; ok {
			if !a.version.Satisfies(pack.Equal, active.version) {
				return nil, fmt.Errorf(
					"Conflicting versions activated: %v (%v, %v)",
					a.Name, a.version, version)
			}
		} else {
			dedupActs[a.Name] = a
		}
	}
	if verbose {
		log.Println(activations)
		log.Println(dedupActs)
	}
	return dedupActs, nil
}
