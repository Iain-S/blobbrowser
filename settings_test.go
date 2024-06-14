package main

import "testing"

func TestGetSettings(t *testing.T) {
	requireEnv = func(_ string) string {
		return "xy"
	}
	getEnv = func(_ string) string {
		return "true"
	}
	settings := GetSettings()
	if !(settings.accountName == "xy" &&
		settings.containerName == "xy" &&
		settings.defaultCredential &&
		settings.secret == "xy") {
		t.Errorf("Error: %+v", settings)
	}
}

func TestRequireEnvFunc(t *testing.T) {
	called := false
	fatalf = func(_ string, _ ...interface{}) {
		called = true
	}
	requireEnvFunc("unlikelykey")
	if !called {
		t.Error()
	}
}
