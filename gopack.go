/*
Package gopack is the command line utility to interact with gopacks repositories
and manage dependencies for projects.
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
 init     - Create a packfile.yaml for the current package.
 pack     - Install the dependencies for the current package.
 packset  - Use a specific packset, will create it if it doesn't exist.

Additional Help: http://gopacks.org/getstarted`
)

func main() {
	var err error

	// Set paths
	err = setPaths()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Ensure we have a configuration.
	created, err := ensureConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else if created {
		fmt.Println("Created configuration file:", gopackConfigPath)
	} else {
		err = loadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if len(os.Args) == 1 {
		fmt.Println(USAGE)
		return
	}

	switch os.Args[1] {
	case "init":
		err = initPackage(PACKFILE, os.Args[2:], os.Stdin, os.Stdout)
	case "pack":
	case "packset":
		err = setPackset(os.Args[2:], os.Stdout)
		if err != nil {
			break
		}
		err = saveConfig()
	default:
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
