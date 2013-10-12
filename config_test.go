package main

import (
	"bytes"
	. "testing"
)

var testConfig = "currentset: testset\n"

func Test_LoadConfig(t *T) {
	err := loadConfigReader(bytes.NewBufferString(testConfig))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if config.CurrentSet != "testset" {
		t.Error("Did not deserialize properly.")
	}
}

func Test_SaveConfig(t *T) {
	var buf bytes.Buffer
	config.CurrentSet = "testset"
	err := saveConfigWriter(&buf)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if str := buf.String(); str != testConfig {
		t.Errorf("Expected:\n%s\ngot:\n%s", testConfig, str)
	}
}
