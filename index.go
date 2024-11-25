package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
)

//go:embed *.tmpl
var files embed.FS

var tmpl = template.Must(template.ParseFS(files, "*.tmpl"))

var generator = sync.OnceValue[string](func() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi.Path
	}
	return ""
})

func ServeIndex(w http.ResponseWriter, r *http.Request, l Links) {
	idx := struct {
		Links     Links
		Generator string
		BaseURL   string
	}{
		l.Links(),
		generator(),
		*baseURL,
	}
	w.Header().Set("Cache-Control", linkCacheControl)
	if err := tmpl.ExecuteTemplate(w, "index.html.tmpl", &idx); err != nil {
		log.Printf(err.Error())
		http.Error(w, "<!-- Template error. Check server logs. -->", http.StatusInternalServerError)
	}
}
