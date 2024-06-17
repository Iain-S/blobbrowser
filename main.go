package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
)

// Variables that can be overridden for testing.
var (
	getServer = getServerFunc
	logFatal  = func(e error) { log.Fatal(e) }
)

func getServerFunc(mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:         ":80",
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
		Handler:      handlers.LoggingHandler(os.Stdout, mux),
	}
}

// Start a webserver and listen on port 80.
func main() {
	mux := http.NewServeMux()
	mux.Handle(
		"/",
		http.TimeoutHandler(
			http.HandlerFunc(
				RenderTemplate("login.html", nil),
			),
			1*time.Second,
			"<html><body>Request timeout!</body></html>\n",
		),
	)
	mux.Handle(
		"/list",
		http.TimeoutHandler(
			http.HandlerFunc(
				GetHomePage(
					GetSettings(),
				),
			),
			1*time.Second,
			"<html><body>Request timeout!</body></html>\n",
		),
	)
	srv := getServer(mux)
	slog.Info("Starting server...")
	logFatal(
		srv.ListenAndServe(),
	)
}
