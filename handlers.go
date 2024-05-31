package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
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
	accountName, ok := lookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		fatal("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}
	containerName, ok := lookupEnv("AZURE_CONTAINER_NAME")
	if !ok {
		fatal("AZURE_CONTAINER_NAME could not be found")
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	slog.Info("Creating Azure credential.")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	// use a managed identity credential in production
	// cred, err := azidentity.NewManagedIdentityCredential(nil)
	if err != nil {
		panic(err)
	}

	slog.Info("Creating Azure Blob Storage client.")
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		panic(err)
	}

	pager := client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{Deleted: true, Versions: true},
	})
	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net/", accountName),
		cred,
		&service.ClientOptions{},
	)
	// Set current and past time and create key
	now := time.Now().UTC().Add(-10 * time.Second)
	expiry := now.Add(48 * time.Hour)
	info := service.KeyInfo{
		Start:  to.Ptr(now.UTC().Format(sas.TimeFormat)),
		Expiry: to.Ptr(expiry.UTC().Format(sas.TimeFormat)),
	}
	udc, err := svcClient.GetUserDelegationCredential(context.Background(), info, nil)
	if err != nil {
		panic(err)
	}

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
			sasQueryParams, err := sas.BlobSignatureValues{
				Protocol:   sas.ProtocolHTTPS,
				StartTime:  time.Now().UTC().Add(time.Second * -10),
				ExpiryTime: time.Now().UTC().Add(15 * time.Minute),
				// Permissions:   to.Ptr(sas.ContainerPermissions{Read: true, List: true}).String(),
				Permissions:   to.Ptr(sas.BlobPermissions{Read: true}).String(),
				ContainerName: containerName,
			}.SignWithUserDelegation(udc)
			if err != nil {
				panic(err)
			}

			sasURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", accountName, containerName, *(_blob.Name), sasQueryParams.Encode())
			// accessTier := *(_blob.Properties.AccessTier)
			// accessTierString := string(accessTier)
			mapBlobs[*(_blob.Name)] = sasURL
		}
	}

	closure := func(w http.ResponseWriter, r *http.Request) {
		timerStart := time.Now()
		f(w, r, mapBlobs)
		slog.Info("Request took", slog.String("t", time.Since(timerStart).String()))
	}

	return closure
}
