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
	http.DefaultServeMux.Handle(
		"/",
		http.TimeoutHandler(
			http.HandlerFunc(
				ServeStaticPage("login.html", nil),
			),
			1*time.Second,
			"<html><body>Request timeout!</body></html>\n",
		),
	)
	settings := GetSettings()
	credentials := GetCredentials(settings.defaultCredential)
	http.DefaultServeMux.Handle(
		"/list",
		http.TimeoutHandler(
			http.HandlerFunc(
				GetHomePage(
					settings,
					credentials,
				),
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
