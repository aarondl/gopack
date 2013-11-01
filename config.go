package main

import (
	"github.com/aarondl/pack"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path/filepath"
)

const (
	CONFIGFILE = "config.yaml"
	DEFAULTSET = "default"
)

// config is the global configuration variable.
var config Configuration

// Configuration holds the configuration for the gp tool.
type Configuration struct {
	CurrentSet string
}

// ensureConfig ensures that a configuration file is present. Returns true
// if a new config was created.
func ensureConfig() (bool, error) {
	_, err := os.Stat(filepath.Join(PATHS.GopackPath, CONFIGFILE))
	if err == nil {
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	_, err = pack.EnsureDirectory(PATHS.GopackPath)
	if err != nil {
		return false, err
	}
	config.CurrentSet = DEFAULTSET
	err = saveConfig()
	return true, err
}

// loadConfig loads the configuration.
func loadConfig() error {
	file, err := os.Open(filepath.Join(PATHS.GopackPath, CONFIGFILE))
	if err != nil {
		return err
	}
	defer file.Close()
	err = loadConfigReader(file)
	if err != nil {
		return err
	}
	return nil
}

// loadConfigReader loads the global configuration object from a reader.
func loadConfigReader(in io.Reader) error {
	all, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	err = goyaml.Unmarshal(all, &config)
	return err
}

// saveConfig writes the configuration.
func saveConfig() error {
	file, err := os.Create(filepath.Join(PATHS.GopackPath, CONFIGFILE))
	if err != nil {
		return err
	}
	defer file.Close()
	err = saveConfigWriter(file)
	return err
}

// saveConfigWriter writes the global configuration object to a writer.
func saveConfigWriter(out io.Writer) error {
	all, err := goyaml.Marshal(&config)
	if err != nil {
		return err
	}
	_, err = out.Write(all)
	return err
}
