package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

type QueryStatus struct {
	str            string
	expectedStatus int
}

func TestHome(t *testing.T) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(
		PasswordProtect(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			"password",
		),
	)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestServeStatic(t *testing.T) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RenderTemplate("login.html", nil))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestAllowGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tt := range []QueryStatus{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
	} {
		t.Run(
			tt.str,
			func(qs QueryStatus) func(*testing.T) {
				return func(t *testing.T) {
					t.Parallel()
					req, err := http.NewRequestWithContext(ctx, qs.str, "/", http.NoBody)
					if err != nil {
						t.Fatal(err)
					}

					rr := httptest.NewRecorder()
					handler := http.HandlerFunc(
						AllowGet(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusOK)
						}),
					)

					handler.ServeHTTP(rr, req)
					expected := http.StatusOK
					if status := rr.Code; status != qs.expectedStatus {
						t.Errorf("handler returned wrong status code: got %v want %v",
							status, expected)
					}
				}
			}(tt),
		)
	}
}

func TestPasswordProtect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tt := range []QueryStatus{
		{"/?_passwordx=qwerty", http.StatusOK},
		{"/?_passwordx=wrong-pass", http.StatusUnauthorized},
		{"/", http.StatusUnauthorized},
	} {
		t.Run(
			tt.str,
			func(qs QueryStatus) func(*testing.T) {
				return func(t *testing.T) {
					t.Parallel()

					req, err := http.NewRequestWithContext(
						ctx,
						http.MethodGet,
						qs.str,
						http.NoBody,
					)
					if err != nil {
						t.Fatal(err)
					}

					rr := httptest.NewRecorder()
					handler := http.HandlerFunc(
						PasswordProtect(
							func(w http.ResponseWriter, _ *http.Request) {
								w.WriteHeader(http.StatusOK)
							},
							// hashed "qwerty"
							"$2y$10$RgvwyipsCjwA5LmTCOcCQO0m.2iucAiLfuc/GodWNP3nTPYCEmoNe",
						),
					)

					handler.ServeHTTP(rr, req)
					if status := rr.Code; status != qs.expectedStatus {
						t.Errorf("handler returned wrong status code: got %v want %v",
							status, qs.expectedStatus)
					}
				}
			}(tt),
		)
	}
}

type MockSigValues struct {
	ExpiryTime    time.Time
	ContainerName string
}

func (me MockSigValues) SignWithUserDelegation(
	_ *service.UserDelegationCredential,
) (sas.QueryParameters, error) {
	values := url.Values{}
	values["se"] = []string{me.ExpiryTime.Format(sas.TimeFormat)}
	params := sas.NewQueryParameters(
		values,
		true,
	)
	return params, nil
}

type MockClient struct {
	info service.KeyInfo
}

func (me *MockClient) GetUserDelegationCredential(
	_ context.Context,
	info service.KeyInfo,
	_ *service.GetUserDelegationCredentialOptions,
) (*service.UserDelegationCredential, error) {
	me.info = info
	udc := service.UserDelegationCredential{}
	return &udc, nil
}

func TestGetEncodedParams(t *testing.T) {
	GetSignatureValues = func(
		expiryTime time.Time,
		containerName string,
	) signatureValuesInterface {
		return MockSigValues{
			ExpiryTime:    expiryTime,
			ContainerName: containerName,
		}
	}

	mockClient := &MockClient{}
	GetServiceClient = func(
		_ string,
		_ azcore.TokenCredential,
	) serviceClientInterface {
		return mockClient
	}
	params := GetEncodedParams("http://localhost", nil, "containerName")
	expected := "se=" + url.QueryEscape(*mockClient.info.Expiry)
	if expected != params {
		t.Error("Client and SAS expiry times don't match")
	}
	expected = "se=" + url.QueryEscape(
		time.Now().UTC().Add(-10*time.Second).Add(3*7*24*time.Hour).Format(sas.TimeFormat),
	)
	if expected != params {
		t.Error("Client and SAS expiry times should be three weeks from ten seconds ago")
	}
}
