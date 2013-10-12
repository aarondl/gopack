package main

import (
	"os"
	"path/filepath"
	. "testing"
)

func Test_SetPaths(t *T) {
	store := os.Getenv(GOPATH)
	defer func() {
		os.Setenv(GOPATH, store)
		setPaths()
	}()

	fakeGoPath := "/tmp:/usr"
	os.Setenv(GOPATH, fakeGoPath)
	config.CurrentSet = DEFAULTSET

	err := setPaths()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	expect := fakeGoPath
	if expect != gopath {
		t.Errorf("Expected: %s, got: %s", expect, gopath)
	}
	expect = filepath.Join("/tmp/", GOPACKFOLDER)
	if expect != gopackPath {
		t.Errorf("Expected: %s, got: %s", expect, gopackPath)
	}
	expect = filepath.Join("/tmp/", GOPACKFOLDER, CONFIGFILE)
	if expect != gopackConfigPath {
		t.Errorf("Expected: %s, got: %s", expect, gopackPath)
	}
	expect = filepath.Join("/tmp/", GOPACKFOLDER, config.CurrentSet)
	if expect != gopacksetPath {
		t.Errorf("Expected: %s, got: %s", expect, gopackPath)
	}
	expect = fakeGoPath + ":/tmp/" + GOPACKFOLDER + "/" + config.CurrentSet
	if expect != combinedPath {
		t.Errorf("Expected: %s, got: %s", expect, combinedPath)
	}
}

func Test_EnsureDirectory(t *T) {
	if Short() {
		t.SkipNow()
	}
	tmp := os.TempDir()
	testdir := filepath.Join(tmp, "newtest")

	created, err := ensureDirectory(testdir)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if !created {
		t.Error("Expected the folder to be created.")
	}

	_, err = os.Stat(testdir)
	if os.IsNotExist(err) {
		t.Error("Expected the folder to be created.")
	}

	created, err = ensureDirectory(testdir)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if created {
		t.Error("Expected the folder to exist.")
	}

	os.Remove(testdir)
}

func Test_GopathSetRestore(t *T) {
	store := os.Getenv(GOPATH)
	defer os.Setenv(GOPATH, store)

	gopathSet()
	if os.Getenv(GOPATH) != combinedPath {
		t.Error("Expected the combined path to have been set.")
	}
	gopathRestore()
	if os.Getenv(GOPATH) != store {
		t.Error("Expected the original path to have been restored.")
	}
}

func Test_TryUriParse(t *T) {
	uri, err := tryUriParse(`/path/to/file`)
	if uri != nil {
		t.Error("Expected nil url on file path, got:", uri)
	}

	uri, err = tryUriParse(`git://github.com/aarondl/pack`)
	if err != nil {
		t.Error("Expected no error, got:", err)
	}
	if uri == nil {
		t.Error("Uri should not be nil.")
	}

	uri, err = tryUriParse(`ssh+git://github.com/aarondl/pack`)
	if err != nil {
		t.Error("Expected no error, got:", err)
	}
	if uri == nil {
		t.Error("Uri should not be nil.")
	}

	uri, err = tryUriParse(`bad/path`)
	if err == nil {
		t.Error("Expected error, but it was nil.")
	}
}
