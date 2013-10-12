package main

import (
	"fmt"
	"io"
)

// setPackset sets the current packset, creating the directory if necessary.
func setPackset(args []string, out io.Writer) error {
	if len(args) == 0 {
		return getPackset(out)
	}

	config.CurrentSet = args[0]
	newpath := makePacksetPath()
	created, err := ensureDirectory(newpath)
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
