package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

const (
	GOPATH       = "GOPATH"
	GOPACKFOLDER = "gopack"
	SRCFOLDER    = "src"
)

var (
	errGoPathNotSet  = errors.New("GOPATH must be set to use this tool.")
	gopath           string
	gopackPath       string
	gopacksetPath    string
	combinedPath     string
	gopackConfigPath string
)

// setPaths uses the environment to calculate all the paths to be used.
func setPaths() error {
	gopath = os.Getenv(GOPATH)
	if len(gopath) == 0 {
		return errGoPathNotSet
	}
	paths := filepath.SplitList(gopath)
	gopackPath = filepath.Join(paths[0], GOPACKFOLDER)
	gopackConfigPath = filepath.Join(gopackPath, CONFIGFILE)
	gopacksetPath = makePacksetPath()
	combinedPath = gopath + string(filepath.ListSeparator) + gopacksetPath
	return nil
}

// makePacksetPath gets the current packset's directory.
func makePacksetPath() string {
	return filepath.Join(gopackPath, config.CurrentSet, SRCFOLDER)
}

// ensureDirectory ensures a directory exists, or it creates it. Returns
// true if the directory had to be created.
func ensureDirectory(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0770)
	}
	return true, err
}

// gopathRestore restores the original gopath variable.
func gopathRestore() {
	os.Setenv(GOPATH, gopath)
}

// gopathAppend adds the combined path to the current gopath.
func gopathSet() {
	os.Setenv(GOPATH, combinedPath)
}

// tryUriParse tries to parse the given string into a uri.
func tryUriParse(pathOrUrl string) (*url.URL, error) {
	if filepath.IsAbs(pathOrUrl) {
		return nil, nil
	}
	url, err := url.ParseRequestURI(pathOrUrl)
	if err != nil {
		return nil, err
	}
	if !url.IsAbs() {
		return nil, fmt.Errorf(`Expected "%s", to be an absolute path or url.`,
			pathOrUrl)
	}
	return url, nil
}
