package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHome(t *testing.T) {
	lookupEnv = func(_ string) (string, bool) {
		return "test", true
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetListBlobs())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestGetListBlobs(_ *testing.T) {
	lookupEnv = func(_ string) (string, bool) {
		return "test", true
	}
	fatal = func(_ ...interface{}) {
	}

	listFunc := GetListBlobs()
	listFunc(*(new(http.ResponseWriter)), &http.Request{})
}
