package main

import (
	"fmt"
	"github.com/aarondl/pack"
	"io"
)

// setPackset sets the current packset, creating the directory if necessary.
func setPackset(args []string, out io.Writer) error {
	if len(args) == 0 {
		return getPackset(out)
	}

	config.CurrentSet = args[0]
	PATHS.SetPackset(config.CurrentSet)
	created, err := pack.EnsureDirectory(PATHS.GopacksetPath)
	if err != nil {
		return err
	}

	if created {
		fmt.Fprintln(out, "Created new packset:", config.CurrentSet)
	}
	fmt.Fprintln(out, "Switched current packset to:", config.CurrentSet)

	return nil
}

// getPackset views the current packset.
func getPackset(out io.Writer) error {
	_, err := fmt.Fprintln(out, config.CurrentSet)
	return err
}
