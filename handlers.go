package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"golang.org/x/crypto/bcrypt"
)

// Data to render the home page.
type TemplateData struct {
	Blobs map[string]BlobInfo
	Title string
}

// Info about an Azure blob.
type BlobInfo struct {
	URL  string
	Size string
}

// Write a buffer back to the client.
func WriteBufferToResponse(
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

// Return a handler function that writes a static page to the response.
func ServeStaticPage(templateName string, data any) func(http.ResponseWriter, *http.Request) {
	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	t := template.Must(template.ParseFiles(templateName))
	err := t.Execute(
		foo,
		data,
	)
	if err != nil {
		panic(err)
	}
	err = foo.Flush()
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		WriteBufferToResponse(w, r, &b)
	}
}

// Password protect a handler function with a secret.
func PasswordProtect(
	f func(http.ResponseWriter, *http.Request),
	secret string,
) func(
	w http.ResponseWriter,
	r *http.Request,
) {
	closure := func(w http.ResponseWriter, r *http.Request) {
		password := r.URL.Query().Get("_passwordx")
		if password == "" {
			http.Error(w, "No password supplied", http.StatusUnauthorized)
			return
		}

		err := bcrypt.CompareHashAndPassword([]byte(secret), []byte(password))
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		timerStart := time.Now()
		f(w, r)
		slog.Info("Request took", slog.String("t", time.Since(timerStart).String()))
	}
	return closure
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

// Get a list of blobs from Azure Blob Storage.
func GetListBlobs(s Settings) func(http.ResponseWriter, *http.Request) {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", s.accountName)

	slog.Info("Creating Azure credential.")
	var cred azcore.TokenCredential
	var err error
	if s.defaultCredential {
		// Note, use a default credential locally as there will be no managed identity.
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	} else {
		// Note, use a managed identity credential in production to avoid timeouts.
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
	containerClient := client.ServiceClient().NewContainerClient(s.containerName)

	pager := containerClient.NewListBlobsHierarchyPager(
		"",
		&container.ListBlobsHierarchyOptions{
			Include:    container.ListBlobsInclude{},
			MaxResults: to.Ptr(int32(1)), // MaxResults set to 1 for demonstration purposes
		},
	)

	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net/", s.accountName),
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
				ContainerName: s.containerName,
			}.SignWithUserDelegation(udc)
			if err != nil {
				panic(err)
			}

			sasURL := fmt.Sprintf(
				"https://%s.blob.core.windows.net/%s/%s?%s",
				s.accountName,
				s.containerName,
				*(_blob.Name),
				sasQueryParams.Encode(),
			)
			mapBlobs[*(_blob.Name)] = BlobInfo{
				sasURL,
				ByteCountIEC(*_blob.Properties.ContentLength),
			}
		}
	}

	return PasswordProtect(
		ServeStaticPage(
			"home.html",
			TemplateData{mapBlobs, "My Blobs"},
		),
		s.secret,
	)
}
