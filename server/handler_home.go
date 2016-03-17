package server

import (
	"log"
	"net/http"
)

type HomeHandler struct{}

func (h HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		log.Printf("not found: %q", r.URL.String())
		http.NotFound(w, r)
		return
	}
	if !AllowMethods(w, r, GET) {
		return
	}
	var page Page
	RenderHTML(w, r, "home", &page, CacheControlPrivate)
}
