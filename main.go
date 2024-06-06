package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
)

// Start a webserver and listen on port 80.
func main() {
	srv := &http.Server{
		Addr:         ":80",
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
		Handler:      handlers.LoggingHandler(os.Stdout, http.DefaultServeMux),
	}
	// some trivial change
	http.DefaultServeMux.Handle(
		"/",
		http.TimeoutHandler(
			http.HandlerFunc(
				Login,
			),
			1*time.Second,
			"<html><body>Request timeout!</body></html>\n",
		),
	)
	http.DefaultServeMux.Handle(
		"/list",
		http.TimeoutHandler(
			http.HandlerFunc(
				GetListBlobs(Home),
			),
			1*time.Second,
			"<html><body>Request timeout!</body></html>\n",
		),
	)
	slog.Info("Starting server...")
	log.Fatal(
		srv.ListenAndServe(),
	)
}
