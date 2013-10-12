package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/aarondl/pack"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	errNoImportPath = errors.New("Could not determine the package import path.")
)

// initPackage first gathers default data (or should)
// and then gathers the rest of the data from the user via console.
func initPackage(file string, args []string,
	in io.Reader, out io.Writer) error {

	var p pack.Pack
	var err error
	s := bufio.NewScanner(in)

	var wd string
	wd, err = os.Getwd()
	if err != nil {
		return err
	}

	_, err = os.Stat(file)
	if err == nil {
		return err
	}

	fmt.Fprintf(out, "Creating initial packfile...")

	// Get package name
	p.Name = path.Base(wd)
	getInput(s, out, "Name", &p.Name)

	// Get import path
	p.ImportPath, err = getImportPath(wd)
	if err != nil {
		return err
	}
	getInput(s, out, "Import Path", &p.ImportPath)

	// Misc Details
	getInput(s, out, "Summary", &p.Summary)
	getInput(s, out, "Description", &p.Description)
	getInput(s, out, "Homepage", &p.Homepage)
	p.License = "MIT"
	getInput(s, out, "License", &p.License)

	err = p.WritePackFile(file)
	if err != nil {
		return err
	}

	return nil
}

// getInput displays a prompt with a default input. If the user enters anything
// the value is replaced by that.
func getInput(in *bufio.Scanner, out io.Writer, name string, value *string) {
	fmt.Fprintf(out, "%s [%s]: ", name, *value)
	if in.Scan() {
		if t := in.Text(); len(t) > 0 {
			*value = t
		}
	}
}

// getImportPath retrieves an import path of a package.
func getImportPath(fullpath string) (importPath string, err error) {
	gopath := os.Getenv(GOPATH)
	gopaths := filepath.SplitList(gopath)
	for i := 0; i < len(gopaths); i++ {
		gopath = path.Join(gopaths[0], "src")
		if index := strings.Index(fullpath, gopath); index >= 0 {
			importPath = fullpath[len(gopath)+1:]
		}
	}

	if len(importPath) == 0 {
		err = errNoImportPath
	}
	return
}
