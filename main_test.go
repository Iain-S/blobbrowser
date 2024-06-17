package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestMain(t *testing.T) {
	requireEnv = func(_ string) string {
		return "xy"
	}
	getServer = func(mux *http.ServeMux) *http.Server {
		server := getServerFunc(mux)
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Errorf("got:%v, want:nil", err)
		}
		return server
	}
	GetHomePage = func(_ Settings) func(w http.ResponseWriter, _ *http.Request) {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	var e error
	logFatal = func(err error) {
		e = err
	}

	main()
	if !errors.Is(e, http.ErrServerClosed) {
		panic("error")
	}
}
