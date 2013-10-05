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
// and an easy way to do cumulative versioning.
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
// the array returned must be in sorted order for the best result from the
// solver as it assumes [0] is the latest version, and [1]... is less than that.
//
// GetVersions: Get the versions only from the dependency information.
// GetGraphs: Get a list of graphs showing dependency information for each
// version of the package.
type versionProvider interface {
	GetVersions(string) []*pack.Version
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
	var noversions bool
	var filter bitFilter
	var ok bool
	var conflicts = make([]string, 0)

	var verbose = true // Replace by flag.

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
		noversions = true
		for j := 0; j < len(vs); j++ {
			for _, con := range current.d.Constraints {
				if !vs[j].Satisfies(con.Operator, con.Version) {
					if verbose {
						log.Println("Removing unacceptable version:", vs[j])
					}
					filter.Set(uint(j))
				} else {
					noversions = false
				}
			}
		}

		if noversions {
			if parent == g.head {
				return fmt.Errorf("No versions to satisfy root dependency: %v", current.d)
			} else {
				// Jump up the stack, conflict, something!
				return fmt.Errorf("No versions satisfy: %v", current.d)
			}
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
		noversions = false
		if active != nil {
			if verbose {
				log.Println("Found activation:", active)
			}

			// Check that we comply with the currently active.
			for _, con := range current.d.Constraints {
				if !active.version.Satisfies(con.Operator, con.Version) {
					// We've found a problem.
					log.Printf("Conflict (%v): %v fails new constraint: %v%v",
						name, active.version, con.Operator.String(),
						con.Version)

					conflicts = append(conflicts, name)
				}
			}

			// If there's a conflict
			if len(conflicts) > 0 {
				if parent == g.head {
					return fmt.Errorf("Reached the top of the stack!")
				}

				// We can still climb the stack, try it.
				sn := stack[si]
				parent = sn.parent
				current = sn.current
				if verbose {
					log.Println("Popping:", current.d.Name)
				}
				ai = sn.ai
				vi = sn.version
				kid = 0
				stack = stack[:len(stack)-1]
				si--
				vi++
				if verbose {
					log.Println(activations)
					log.Println(activations[:ai])
					log.Println("Snipping activations back to:", ai) // TODO: REMOVE
				}
				activations = activations[:ai]
				continue
			}

			// Add the current filter to the primary activation.
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
				// No version could be found, this is a conflict of sorts.
				return fmt.Errorf("No versions could be found for: %v", current.d)
			}
		}

		// Add ourselves to the list of activators.
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
				stack:   nil,
			},
		})
		ai++
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

		// Try activating:
		// If activated:
		//  If conflict:
		//    backjump to first activated parent?
		// Else:
		//  Create activation with list of versions/dependencies.
		// Remove non-compatible versions with the current node.
		// Choose highest version as current activated.
	}

	return nil
}

/*
	var verbose = true // Move to flag

	var stack = make([]stacknode, 0, initialStackSize) // Avoid allocations
	var index = 0
	var kid, version int
	var backjump *savestate
	var current *depnode
	var vs []*pack.Version
	var active = make(map[string]*activation)

	current = g.head

	for i := 0; i < 40; i++ {
		if verbose {
			log.Printf("Eval: %s (%v, %v)\n", current.d.Name, kid, version)
		}

		// Have we run out of dependencies to resolve?
		if kid >= len(current.kids) {
			if verbose {
				log.Println("Ran out of children...")
			}
			// Jump up stack.
			if index > 0 {
				index--
				current = stack[index].depnode
				//version = stack[index].version
				stack[index].kid++
				kid = stack[index].kid
				if verbose {
					log.Printf("Pop: %s (%v, %v)\n", current.d.Name,
						kid, version)
				}
				continue
			}
			if verbose {
				log.Println("Nothing left to do, should be solved.")
			}
			// We did it!!!!
			return true
		}

		// Try to activate child
		curkid := current.kids[kid]
		name := curkid.d.Name
		log.Println("Attempting child activation:", name)

		// Check if already activated
		if act, ok := active[name]; ok && act.v != nil {
			var found bool
			for _, con := range curkid.d.Constraints {
				if act.v.Satisfies(con.Operator, con.Version) {
					found = true
					break
				}
			}

			if found {
				// Add ourselves to the activators list.
				save := &savestate{
					kid, version, index, current,
					make([]stacknode, index+1),
				}
				copy(save.stack, stack)
				act.states = append(act.states, save)

				if verbose {
					log.Println(name, "satisfied by previous activation:",
						act.v, act.states)
				}
			} else {
				// Backjump
				backjump = act.states[len(act.states)-1]
				if verbose {
					log.Printf("Found conflict: %s (%v)\n", name, act.v)
					log.Print("Previous states: ")
					for _, a := range act.states {
						log.Printf("%v ", a)
					}
				}
				act.states = act.states[:len(act.states)-1]
				act.v = nil

				current, kid, version, index = backjump.current,
					backjump.kid, backjump.version, backjump.index
				copy(stack, backjump.stack)
				if verbose {
					log.Println("Backjumping:", index, kid)
				}
				version++
				continue
			}
		}

		// Get versions
		vs = vp.GetVersions(name)
		if verbose {
			log.Printf("Versions: %s %v\n", name, vs)
			log.Println("Iterating from:", version)
		}
		// Each version
		for ; version < len(vs); version++ {
			// Each constraint
			ver := vs[version]
			for _, con := range curkid.d.Constraints {
				if ver.Satisfies(con.Operator, con.Version) {
					if verbose {
						log.Println("Satisfactory Version:", curkid.d, ver)
					}
					curkid.v = ver
				}
				if curkid.v != nil {
					log.Println("No need for more constraint checks... Breaking.")
					break
				}
			}
			if len(curkid.d.Constraints) == 0 {
				curkid.v = ver
			}
			if curkid.v != nil {
				log.Println("No need to check more versions... Breaking")
				break
			}
		}

		if curkid.v == nil {
			// No version found to satisfy.
			if verbose {
				log.Println("No versions available to satisfy:", curkid.d)
			}
			return false
		} else {
			// Activate
			if verbose {
				log.Printf("Activating: %s %v (%v, %v)\n",
					curkid.d.Name, curkid.v, version, index)
			}

			// Add save state info to activation
			save := &savestate{
				kid, version, index, current,
				make([]stacknode, index+1),
			}
			copy(save.stack, stack)

			if act, ok := active[name]; ok {
				act.states = []*savestate{save}
				act.v = curkid.v
			} else {
				active[name] = &activation{[]*savestate{save}, curkid.v}
			}

			// Pull in child dependencies on activation.
			graphs := vp.GetGraphs(name)
			found := false
			for i := 0; i < len(graphs); i++ {
				if graphs[i].head.v.Satisfies(pack.Equal, curkid.v) {
					found = true
					curkid.kids = graphs[i].head.kids
					if verbose {
						log.Println("Setting kids:", len(curkid.kids))
					}
					break
				}
			}

			if !found {
				if verbose {
					log.Println("Dependency graph missing for: %v %v",
						curkid.d.Name, curkid.v)
				}
				return false
			}
		}

		if len(curkid.kids) > 0 {
			// Push current on to stack, make the curkid the new current.
			if verbose {
				log.Println("Has kids:", len(curkid.kids))
				log.Printf("Push: %s (%v, %v)\n", current.d.Name, kid, version)
			}
			stack = append(stack, stacknode{kid, version, current})
			kid = 0
			version = 0
			current = curkid
			index++
		} else {
			// Continue through all kids.
			kid++
			if verbose {
				log.Println("Next kid:", kid)
			}
		}
	}

	return false
}*/
