package main

import (
	"bufio"
	"bytes"
	"github.com/aarondl/pack"
	"os"
	. "testing"
)

func TestInit(t *T) {
	var input = bytes.NewBufferString("\n\n\n\n\n\n\n\n")
	var devnull bytes.Buffer
	var err error
	PATHS, err = pack.NewPathsFromGopath(DEFAULTSET)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	file := "test.yaml"
	if err := initPackage(file, nil, input, &devnull); err != nil {
		t.Error("Unexpected Error:", err)
	}

	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		t.Errorf("Expected %s to exist but got:", err)
	}

	p, err := pack.ParsePackFile(file)
	if err != nil {
		t.Fatal("Expected the serialized file to be deserializable.")
	}

	if len(p.Name) == 0 || len(p.ImportPath) == 0 {
		t.Error("Expected properties to get default values.")
	}

	os.Remove(file)
}

func TestInit_GetInput(t *T) {
	var str string
	var devnull bytes.Buffer
	var buf = bytes.NewBufferString("\n\ninput\n")
	var scanner = bufio.NewScanner(buf)

	getInput(scanner, &devnull, "Test", &str)
	if len(str) > 0 {
		t.Error("String should be empty, got:", str)
	}

	str = "test"
	getInput(scanner, &devnull, "Test", &str)
	if str != "test" {
		t.Error("String should be 'test', got:", str)
	}

	getInput(scanner, &devnull, "Test", &str)
	if str != "input" {
		t.Error("String should be 'input', got:", str)
	}
}

func TestInit_GetImportPath(t *T) {
	var err error
	PATHS, err = pack.NewPathsFromGopath(DEFAULTSET)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	wd, _ := os.Getwd()

	exp := "github.com/aarondl/gopack"
	if path, err := getImportPath(wd); err != nil {
		t.Error("Unexpected error:", err)
	} else if path != exp {
		t.Errorf("Wrong import string, expected: %v got: %v", exp, path)
	}

	if _, err := getImportPath("/"); err != errNoImportPath {
		t.Errorf("Wrong error, expected: %v got: %v", errNoImportPath, err)
	}
}
