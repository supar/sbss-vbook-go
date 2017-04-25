package main

import (
	"flag"
	"net/http"
	"net/http/httputil"
)

func main() {
	var (
		router *Router
		log    *Log
	)

	// Read flags
	flag.Parse()
	// Create logger
	log = NewLogger(NAME, CONSOLELOG)
	// Print version if flag passed
	showVersion(log)

	router = NewRouter(SBSSAPISERVER, log)
	router.watchGarbage()

	router.Handle("PROPFIND", "/.well-known/carddav", func(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
		http.Redirect(w, r, "/carddav", 301)
	})

	router.Handle("PROPFIND", "/carddav/", HandleAuthorize(HandlePropfind))
	router.Handle("PROPFIND", "/carddav/contacts", HandleAuthorize(HandlePropfind))
	router.Handle("REPORT", "/carddav/", HandleAuthorize(HandleReport))
	router.Handle("REPORT", "/carddav/contacts", HandleAuthorize(HandleReport))
	router.Handle("GET", "/carddav/:contact", HandleAuthorize(HandleGetContact))

	// Handle NotFound
	router.HandleMethodNotAllowed = false
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Warn("%s, %s: 404 %s %s", r.RemoteAddr, r.UserAgent(), r.Method, r.URL.RequestURI())

		if data, err := httputil.DumpRequest(r, true); err != nil {
			log.Error(err)
		} else {
			log.Debug("%s", data)
		}

		http.NotFound(w, r)
	})

	log.Notice("Start server at %s", SERVERADDRESS)
	http.ListenAndServe(SERVERADDRESS, router)
}
