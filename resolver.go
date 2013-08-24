package main

import (
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
	GetGraphs(string) []*depgraph
}

// stacknode helps to emulate recursion and perform safejumps.
type stacknode struct {
	kid     int
	version int
	*depnode
}

// activation is the details of a packages activation.
type activation struct {
	// activators is split into two int32's: 0-31 = stackindex, 32-63 = kid
	activators []uint64
	v          *pack.Version
}

// solve solves a dependency graph.
// This algorithm works like this:
// 1. Do depth first search of the graph. (using iteration not recursion)
// 2. Loop through deps, has this been activated?
//  2.l. If yes, can we use it?
//     2.1.1. If yes, activate, add us to the list of activators.
//     2.1.2. If no, backjump to our activation.
//  2.2. If no, backjump to packages last activation spot.
// 3. Profit???
func (g *depgraph) solve(vp versionProvider) bool {
	var verbose = true // Move to flag

	var stack = make([]stacknode, 0, initialStackSize) // Avoid allocations
	var index = 0
	var kid, version int
	var backjump uint64
	var hasConflict bool
	var current *depnode
	var active = make(map[string]activation)

	current = g.head

	for i := 0; i < 20; i++ {
		if verbose {
			log.Printf("Eval: %s (%v, %v)\n", current.d.Name, kid, version)
		}

		hasConflict = false

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
		if act, ok := active[name]; ok {
			for _, con := range curkid.d.Constraints {
				if act.v.Satisfies(con.Operator, con.Version) {
					// Add ourselves to the activators list.
					act.activators = append(act.activators,
						uint64(uint(kid)<<kidOffset|uint(index)))
					if verbose {
						log.Println(name, "satisfied by previous activation:",
							act.v, act.activators)
					}
				} else {
					// Backjump
					backjump = act.activators[len(act.activators)-1]
					hasConflict = true
					if verbose {
						log.Printf("Found conflict: %s (%v)\n", name, act.v)
						log.Print("Previous activators: ")
						for _, a := range act.activators {
							log.Printf("[%v, %v], ",
								a>>kidOffset, a&stackIndexMask)
						}
					}
				}
			}
		}

		if hasConflict {
			//Backjump (avoids using goto, this is dumb)
			index = int(backjump & stackIndexMask)
			kid = int(backjump >> kidOffset)
			if verbose {
				log.Println("Backjumping:", index, kid)
			}
			continue
		}

		// Get versions
		vs := vp.GetVersions(name)
		if verbose {
			log.Printf("Versions: %s %v\n", name, vs)
		}
		if verbose {
			log.Println("Iterating from:", version)
		}
		// Each version
		for ; version < len(vs); version++ {
			// Each constraint
			ver := vs[version]
			for _, con := range curkid.d.Constraints {
				if ver.Satisfies(con.Operator, con.Version) {
					if verbose {
						log.Println("Found a version to satisfy:", curkid.d, ver)
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
			active[name] = activation{[]uint64{
				uint64(uint(kid)<<kidOffset | uint(index))}, curkid.v}

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
}
