package main

import (
	"bytes"
	"github.com/aarondl/pack"
	"os"
	"path/filepath"
	. "testing"
)

func Test_SetPackset(t *T) {
	if Short() {
		t.SkipNow()
	}
	var err error

	testdir := filepath.Join(os.TempDir(), "setpacksettest")
	PATHS, err = pack.NewPaths(testdir, "default")
	config.CurrentSet = DEFAULTSET
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = os.MkdirAll(PATHS.GopacksetPath, 0770)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	defer os.RemoveAll(testdir)

	var buf bytes.Buffer
	err = setPackset([]string{"testset"}, &buf)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	expect1 := "Created new packset: testset\n"
	expect2 := "Switched current packset to: testset\n"
	if str := buf.String(); str != expect1+expect2 {
		t.Errorf("Expected output:\n%s\ngot:\n%s", expect1+expect2, str)
	}

	buf.Reset()
	err = setPackset([]string{"testset"}, &buf)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if str := buf.String(); str != expect2 {
		t.Errorf("Expected output:\n%s\ngot:\n%s", expect2, str)
	}
}

func Test_GetPackset(t *T) {
	var buf bytes.Buffer
	config.CurrentSet = DEFAULTSET
	err := getPackset(&buf)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if str := buf.String(); str != config.CurrentSet+"\n" {
		t.Errorf("Expected %s and a newline, got: %s", config.CurrentSet, str)
	}
}
