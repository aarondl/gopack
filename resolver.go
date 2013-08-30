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
	current *depnode
	parent  *depnode
}

// savestate is the state of the algorithm at an activation point.
type savestate struct {
	kid     int
	version int
	index   int
	current *depnode
	stack   []stacknode
}

// activation is the details of a packages activation.
type activation struct {
	name    string
	version *pack.Version
	states  []*savestate
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
solve a dependency graph.

This algorithm is a depth first search with backjumping to resolve conflicts.

Possible optimization: don't attempt a new version of a package unless it's
dependencies have changed.
*/
func (g *depgraph) solve(vp versionProvider) error {
	if len(g.head.kids) == 0 {
		return nil
	}

	var current, parent *depnode = g.head, nil
	var stack = make([]stacknode, 0, initialStackSize) // Avoid allocations
	var si, kid = -1, 0
	var activations []*activation
	var versions = make(map[string][]*pack.Version)
	var active *activation
	var version *pack.Version
	var vs []*pack.Version
	var ok bool
	var verbose = true // Replace by flag.

	for i := 0; i < 10; i++ {
		name := current.d.Name
		if verbose {
			log.Println("Current:", current.d)
		}

		if current == g.head {
			if kid >= len(current.kids) {
				if verbose {
					log.Println("Success!")
				}
				return nil
			} else {
				goto NEXT
			}
		}

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
		for j := 0; j < len(vs); j++ {
			if vs[j] == nil {
				continue
			}
			for _, con := range current.d.Constraints {
				if !vs[j].Satisfies(con.Operator, con.Version) {
					if verbose {
						log.Println("Removing unacceptable version:", vs[j])
					}
					vs[j] = nil
				}
			}
		}

		// Check for activeness.
		active = nil
		for j := 0; j < len(activations); j++ {
			if activations[j].name == name {
				active = activations[j]
				break
			}
		}

		if verbose && active != nil {
			log.Println("Found activation:", active)
		}

		// Find a suitable version
		version = nil
		for j := 0; version == nil && j < len(vs); j++ {
			if vs[j] == nil {
				continue
			}
			if len(current.d.Constraints) == 0 {
				version = vs[j]
				break
			}
			for _, con := range current.d.Constraints {
				if vs[j].Satisfies(con.Operator, con.Version) {
					version = vs[j]
					break
				}
			}
		}

		if version != nil {
			if verbose {
				log.Printf("Adding: %v %v to activations", name, version)
			}
			// Activate
			if active == nil {
				activations = append(activations, &activation{
					name:    name,
					version: version,
					states: []*savestate{{
						kid:     kid,
						version: 0, //Remove?
						index:   si,
						current: current,
						stack:   nil,
					}},
				})
				current.v = version
			} else {
				active.states = append(active.states, &savestate{
					kid:     kid,
					version: 0, //Remove?
					index:   si,
					current: current,
					stack:   nil,
				})
			}

			if verbose {
				log.Println("Activations:", activations)
			}

			// Fetch dependencies for chosen version.
			if verbose {
				log.Println("Fetching Dependencies for:", name)
			}
			curgraph := vp.GetGraph(name, version)
			current.kids = curgraph.head.kids
			if verbose {
				var b bytes.Buffer
				for _, dep := range current.kids {
					b.WriteString(dep.d.String())
					b.WriteRune(space)
				}
				log.Println("Got dependencies:", b.String())
			}
		} else {
			// No suitable versions.
			return fmt.Errorf("No suitable versions for: %v", current.d)
		}

		// Do the next node
	NEXT:

		// Pop off the stack back to parent.
		if kid < len(current.kids) {
			// Push us on to stack, go into child.
			if verbose {
				log.Println("Pushing:", name, kid)
			}
			stack = append(stack, stacknode{
				kid:     kid,
				version: 0, //remove?
				current: current,
				parent:  parent,
			})
			parent = current
			current = current.kids[kid]
			kid = 0
			si++
			continue
		}

		sn := stack[si]
		parent = sn.parent
		current = sn.current
		if verbose {
			log.Println("Popping:", current.d.Name)
		}
		kid = sn.kid
		kid++
		stack = stack[:len(stack)-1]
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
