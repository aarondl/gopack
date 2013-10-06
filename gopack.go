/*
Package gopack is the command line utility to interact with gopacks repositories
and install dependencies for projects.
*/
package main

import (
	"flag"
)

var DEBUG = flag.Bool("debug", false, "Turns on debug output.")

func main() {
	flag.Parse()
}
