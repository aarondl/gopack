package main

import (
	"bytes"
	"fmt"
	"github.com/aarondl/pack"
	"log"
)

const (
	initialStackSize        = 20
	kidOffset               = 32
	stackIndexMask   uint64 = 0xFFFFFFFF
)

// bitFilter is a type used to filter versions based on their index in an array
type bitFilter uint64

// Set sets a bit in the filter.
func (b bitFilter) Set(index uint) bitFilter {
	return bitFilter((1 << uint64(index)) | uint64(b))
}

// IsSet checks if a a bit in the filter is set.
func (b bitFilter) IsSet(index uint) bool {
	return 0 != ((1 << uint64(index)) & uint64(b))
}

// Clear turns off a bit in the filter.
func (b bitFilter) Clear(index uint) bitFilter {
	return bitFilter(^(1 << uint64(index)) & uint64(b))
}

// Add is the union of two bitFilters.
func (b bitFilter) Add(a bitFilter) bitFilter {
	return bitFilter(uint64(a) | uint64(b))
}

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
	version int
	ai      int
	current *depnode
	parent  *depnode
}

// savestate is the state of the algorithm at an activation point.
type savestate struct {
	kid     int
	version int
	si      int
	ai      int
	current *depnode
	parent  *depnode
	stack   []stacknode
}

// activation is the details of a packages activation.
type activation struct {
	name    string
	version *pack.Version
	state   *savestate
	filter  bitFilter
}

// String is used to debug activations.
func (a activation) String() string {
	var buf bytes.Buffer
	buf.WriteString(a.name)
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
func (g *depgraph) solve(vp versionProvider) error {
	if len(g.head.kids) == 0 {
		return nil
	}

	var current, parent *depnode = g.head, nil
	var stack = make([]stacknode, 0, initialStackSize) // Avoid allocations
	var si, ai, kid = -1, -1, 0
	var activations []*activation
	var active *activation
	var versions = make(map[string][]*pack.Version)
	var version *pack.Version
	var vs []*pack.Version
	var vi int
	var filter bitFilter
	var excluded int
	var ok bool
	var conflicts = make([]string, 0)
	var conflict bool

	var verbose = false // Replace by flag.

	for i := 0; i < 20; i++ {
		name := current.d.Name
		if verbose {
			log.Println("Current:", current.d)
		}

		if current == g.head {
			if kid >= len(current.kids) {
				if verbose {
					log.Println("Success!")
					log.Println(activations)
				}
				return nil
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

		// Weed out versions.
		filter = 0
		excluded = 0
		for j := 0; j < len(vs); j++ {
			for _, con := range current.d.Constraints {
				if !vs[j].Satisfies(con.Operator, con.Version) {
					if verbose {
						log.Println("Removing unacceptable version:", vs[j])
					}
					filter.Set(uint(j))
					excluded++
				}
			}
		}

		if parent == g.head && len(vs) == excluded {
			return fmt.Errorf("No versions to satisfy root dependency: %v",
				current.d)
		}

		// Check for activeness. The first activation will always serve as the
		// main activation point, with the others simply being save points.
		active = nil
		for j := 0; j < len(activations); j++ {
			if activations[j].name == name {
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
						log.Printf("Conflict (%v): %v fails constraint: %v%v",
							name, active.version, con.Operator.String(),
							con.Version)
					}

					conflict = true
					break
				}
			}

			// Add the current filter to the primary activation.
			version = active.version
			active.filter = active.filter.Add(filter)
		} else {
			if verbose {
				log.Println("Not activated:", name)
			}
			// Find a suitable version
			for ; version == nil && vi < len(vs); vi++ {
				if active != nil && active.filter.IsSet(uint(vi)) {
					continue
				}
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
					log.Printf("Conflict (%v): has no usable versions %v",
						name, vs)
				}
				// No version could be found, this is a conflict of sorts.
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
					return fmt.Errorf("We've tried everything mate: %v",
						conflicts)
				}
				name = conflicts[0]
				conflicts = conflicts[1:]
				var st *savestate
				for i := 0; i < len(activations); i++ {
					if activations[i].name == name {
						st = activations[i].state
						break
					}
				}
				if st == nil {
					return fmt.Errorf("Conflict's activation not found: %v %v",
						name, activations)
				}
				current = st.current
				parent = st.parent
				if verbose {
					log.Println("Conflict! Restoring:", current.d.Name)
				}
				ai = st.ai
				vi = st.version
				vi++
				stack = st.stack
				si = st.si
				kid = 0
				activations = activations[:ai]
				if verbose {
					log.Println("Activations:", activations)
				}
				continue
			}

			// We can still climb the stack, try it.
			sn := stack[si]
			parent = sn.parent
			current = sn.current
			if verbose {
				log.Println("Conflict! Popping:", current.d.Name)
			}
			ai = sn.ai
			vi = sn.version
			vi++
			stack = stack[:len(stack)-1]
			si--
			kid = 0
			activations = activations[:ai]
			if verbose {
				log.Println("Activations:", activations)
			}
			continue
		}

		// Add ourselves to the list of activators.
		ai++
		activations = append(activations, &activation{
			name:    name,
			version: version,
			filter:  filter,
			state: &savestate{
				kid:     kid,
				version: vi,
				si:      si,
				ai:      ai,
				current: current,
				parent:  parent,
				stack:   make([]stacknode, len(stack)),
			},
		})
		copy(activations[len(activations)-1].state.stack, stack)
		current.v = version

		if verbose {
			log.Printf("Added: %v %v to activations", name, version)
			log.Println("Activations:", activations)
		}

		if verbose {
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
			stack = append(stack, stacknode{
				kid:     kid,
				version: vi,
				ai:      ai,
				current: current,
				parent:  parent,
			})
			parent = current
			current = current.kids[kid]
			ai = len(activations) - 1
			kid = 0
			vi = 0
			si++
			continue
		}

		// Pop off the stack back to parent.
		sn := stack[si]
		parent = sn.parent
		current = sn.current
		if verbose {
			log.Println("Popping:", current.d.Name)
		}
		kid = sn.kid
		vi = sn.version
		ai = sn.ai
		stack = stack[:len(stack)-1]
		kid++
		si--
	}

	return nil
}
