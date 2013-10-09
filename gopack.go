/*
Package gopack is the command line utility to interact with gopacks repositories
and install dependencies for projects.
*/
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	DEBUG = flag.Bool("debug", false, "Turns on debug output.")
)

const (
	PACKFILE = "package.yaml"

	USAGE = `gp - Go Pack
	
Usage:
 init - Create a packfile.yaml for the current package.
 pack - Install the dependencies for the current package.
 use  - Use a specific packset, will create it if it doesn't exist.

 Additional Help: http://gopacks.org/getstarted`
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println(USAGE)
		return
	}

	switch os.Args[1] {
	case "init":
		initPackage(PACKFILE, os.Args[2:], os.Stdin, os.Stdout)
	case "pack":
	case "use":
	default:
	}
}
