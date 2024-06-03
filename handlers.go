package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// Variables that can be overridden for testing.
var (
	lookupEnv = os.LookupEnv
	fatal     = log.Fatal
)

type TemplateData struct {
	Blobs map[string]BlobInfo
	Title string
}

type BlobInfo struct {
	URL  string
	Size string
}

// Home is the handler for the root path. It writes the list of blobs to the response.
func Home(
	w http.ResponseWriter,
	r *http.Request,
	buffer *bytes.Buffer,
) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := w.Write(buffer.Bytes())
	if err != nil {
		panic(err)
	}
}

// https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// GetListBlobs wraps a handler function with a function that retrieves a list of blobs from Azure Blob Storage.
func GetListBlobs(
	f func(http.ResponseWriter, *http.Request, *bytes.Buffer),
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
	// Note, use a managed identity credential in production to avoid timeouts.
	var cred azcore.TokenCredential
	var err error
	useDefaultCredential, ok := lookupEnv("USE_DEFAULT_CREDENTIAL")
	if ok && useDefaultCredential == "true" {
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	} else {
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	}
	if err != nil {
		panic(err)
	}

	slog.Info("Creating Azure Blob Storage client.")
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		panic(err)
	}
	containerClient := client.ServiceClient().NewContainerClient(containerName)

	pager := containerClient.NewListBlobsHierarchyPager(
		"",
		&container.ListBlobsHierarchyOptions{
			Include:    container.ListBlobsInclude{},
			MaxResults: to.Ptr(int32(1)), // MaxResults set to 1 for demonstration purposes
		},
	)

	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net/", accountName),
		cred,
		&service.ClientOptions{},
	)
	if err != nil {
		panic(err)
	}
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
	// Blob names and their SAS URLs
	mapBlobs := make(map[string]BlobInfo)
	slog.Info("Paging.")
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Error in NextPage", slog.Any("error", err))
			break
		}
		for _, _blob := range resp.Segment.BlobItems {
			sasQueryParams, err := sas.BlobSignatureValues{
				Protocol:      sas.ProtocolHTTPS,
				StartTime:     time.Now().UTC().Add(time.Second * -10),
				ExpiryTime:    time.Now().UTC().Add(15 * time.Minute),
				Permissions:   to.Ptr(sas.BlobPermissions{Read: true}).String(),
				ContainerName: containerName,
			}.SignWithUserDelegation(udc)
			if err != nil {
				panic(err)
			}

			sasURL := fmt.Sprintf(
				"https://%s.blob.core.windows.net/%s/%s?%s",
				accountName,
				containerName,
				*(_blob.Name),
				sasQueryParams.Encode(),
			)
			mapBlobs[*(_blob.Name)] = BlobInfo{
				sasURL,
				ByteCountIEC(*_blob.Properties.ContentLength),
			}
		}
	}

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	// use a http/template to render the list of blobs
	t := template.Must(template.ParseFiles("home.html"))
	err = t.Execute(
		foo,
		TemplateData{
			mapBlobs,
			"My Blobs",
		},
	)
	if err != nil {
		panic(err)
	}
	err = foo.Flush()
	if err != nil {
		panic(err)
	}

	closure := func(w http.ResponseWriter, r *http.Request) {
		timerStart := time.Now()
		f(w, r, &b)
		slog.Info("Request took", slog.String("t", time.Since(timerStart).String()))
	}

	return closure
}
