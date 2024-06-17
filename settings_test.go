package main

import "testing"

func TestGetSettingsMin(t *testing.T) {
	requireEnv = func(_ string) string {
		return "xy"
	}
	getEnv = func(_ string) string {
		return ""
	}
	settings := GetSettings()
	if !(settings.accountName == "xy" &&
		settings.containerName == "xy" &&
		!settings.defaultCredential &&
		settings.secret == "xy" &&
		settings.title == "Blob Browser") {
		t.Errorf("Error: %+v", settings)
	}
}

func TestGetSettingsMax(t *testing.T) {
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
		settings.secret == "xy" &&
		settings.title == "true") {
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
