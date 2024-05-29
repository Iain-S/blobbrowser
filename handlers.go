package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Variables that can be overridden for testing.
var (
	lookupEnv = os.LookupEnv
	fatal     = log.Fatal
)

// Home is the handler for the root path. It writes the list of blobs to the response.
func Home(w http.ResponseWriter, r *http.Request, blobs map[string]string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, blobs)
}

// GetListBlobs wraps a handler function with a function that retrieves a list of blobs from Azure Blob Storage.
func GetListBlobs(
	f func(http.ResponseWriter, *http.Request, map[string]string),
) func(http.ResponseWriter, *http.Request) {
	// Get a list of blobs from Azure Blob Storage
	envVarName := "BLOB_SAS_URL"
	sasToken, ok := lookupEnv(envVarName)
	if !ok {
		fatal(envVarName + " could not be found")
	}

	slog.Info("Creating Azure Blob Storage client.")
	client, err := azblob.NewClientWithNoCredential(sasToken, nil)
	if err != nil {
		panic(err)
	}

	pager := client.NewListBlobsFlatPager("", &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{Deleted: false, Versions: true},
	})

	ctx := context.Background()
	mapBlobs := make(map[string]string)
	slog.Info("Paging.")
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Error in NextPage", slog.Any("error", err))
			break
		}
		for _, _blob := range resp.Segment.BlobItems {
			accessTier := *(_blob.Properties.AccessTier)
			accessTierString := string(accessTier)
			mapBlobs[*(_blob.Name)] = accessTierString
		}
	}

	closure := func(w http.ResponseWriter, r *http.Request) {
		timerStart := time.Now()
		f(w, r, mapBlobs)
		slog.Info("Request took", slog.String("t", time.Since(timerStart).String()))
	}

	return closure
}
