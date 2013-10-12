package main

import (
	"bytes"
	"os"
	"path/filepath"
	. "testing"
)

func Test_SetPackset(t *T) {
	var pushPath = gopackPath
	gopackPath = os.TempDir()
	defer func() {
		gopackPath = pushPath
		os.Remove(filepath.Join(os.TempDir(), "testset"))
	}()

	var buf bytes.Buffer
	err := setPackset([]string{"testset"}, &buf)
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
