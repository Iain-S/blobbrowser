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

var GetHomePage = GetHomePageFunc

// Write a buffer back to the client.
func AllowGet(
	f http.HandlerFunc,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Should allow GET.
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		f(w, r)
	}
}

// Return a handler function that writes a static page to the response.
func RenderTemplate(
	templateName string,
	data any,
) func(http.ResponseWriter, *http.Request) {
	var buffer bytes.Buffer
	foo := bufio.NewWriter(&buffer)
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
	return func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		_, err := w.Write(buffer.Bytes())
		if err != nil {
			panic(err)
		}
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
	return func(w http.ResponseWriter, r *http.Request) {
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
}

// Get a handler function that renders the home page.
func GetHomePageFunc(
	s Settings,
) func(http.ResponseWriter, *http.Request) {
	homePageData := GetHomePageData(
		GetCredentials(s.defaultCredential),
		s.accountName,
		s.containerName,
	)

	return AllowGet(
		PasswordProtect(
			RenderTemplate(
				"home.html",
				TemplateData{homePageData, "My Blobs"},
			),
			s.secret,
		),
	)
}

// Get data to show on the home page.
func GetHomePageData(
	creds azcore.TokenCredential,
	accountName string,
	containerName string,
) map[string]BlobInfo {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobs := GetBlobs(
		serviceURL,
		creds,
		containerName,
	)
	params := GetEncodedParams(
		serviceURL,
		creds,
		containerName,
	)
	mapBlobs := make(map[string]BlobInfo)
	for _, _blob := range blobs {
		sasURL := serviceURL + fmt.Sprintf(
			"%s/%s?%s",
			containerName,
			*(_blob.Name),
			params,
		)
		mapBlobs[*(_blob.Name)] = BlobInfo{
			sasURL,
			ByteCountIEC(*_blob.Properties.ContentLength),
		}
	}
	return mapBlobs
}

// Get a list of blobs from Azure Blob Storage.
func GetBlobs(
	serviceURL string,
	cred azcore.TokenCredential,
	containerName string,
) []*container.BlobItem {
	slog.Info("Creating Azure Blob Storage client.")
	client, err := azblob.NewClient(serviceURL, cred, &azblob.ClientOptions{})
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

	ctx := context.Background()
	blobItems := make([]*container.BlobItem, 0)
	slog.Info("Paging.")
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Error in NextPage", slog.Any("error", err))
			break
		}
		blobItems = append(blobItems, resp.Segment.BlobItems...)
	}

	return blobItems
}

func GetCredentials(
	useDefaultCredential bool,
) azcore.TokenCredential {
	var cred azcore.TokenCredential
	var err error
	slog.Info("Creating Azure credential.")
	if useDefaultCredential {
		// Note, use a default credential locally as there will be no managed identity.
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	} else {
		// Note, use a managed identity credential in production to avoid timeouts.
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	}
	if err != nil {
		panic(err)
	}
	return cred
}

// GetEncodedParams returns the encoded SAS query parameters.
func GetEncodedParams(
	serviceURL string,
	cred azcore.TokenCredential,
	containerName string,
) string {
	svcClient, err := service.NewClient(
		serviceURL,
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
	udc, err := svcClient.GetUserDelegationCredential(
		context.Background(),
		info,
		&service.GetUserDelegationCredentialOptions{},
	)
	if err != nil {
		panic(err)
	}
	// A container-level SAS
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(time.Second * -10),
		ExpiryTime:    time.Now().UTC().Add(15 * time.Minute),
		Permissions:   to.Ptr(sas.ContainerPermissions{Read: true}).String(),
		BlobName:      "",
		ContainerName: containerName,
	}.SignWithUserDelegation(udc)
	if err != nil {
		panic(err)
	}

	return sasQueryParams.Encode()
}
