package main

import (
	"log"
	"os"
)

// Variables that can be overridden for testing.
var (
	fatalf = log.Fatalf
)

type Settings struct {
	accountName       string // The name of the Azure Storage account.
	containerName     string // The name of the Azure Storage container.
	defaultCredential bool   // Use a default Azure credential.
	secret            string // The (hashed) secret to use for authentication.
}

func requireEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		fatalf("Environment variable %s is required", key)
	}
	return value
}

func GetSettings() Settings {
	return Settings{
		accountName:   requireEnv("AZURE_STORAGE_ACCOUNT_NAME"),
		containerName: requireEnv("AZURE_CONTAINER_NAME"),
		defaultCredential: func() bool {
			return os.Getenv("USE_DEFAULT_CREDENTIAL") == "true"
		}(),
		secret: requireEnv("BLOBBROWSER_SECRET"),
	}
}
