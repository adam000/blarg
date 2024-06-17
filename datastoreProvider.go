package blarg

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/datastore"

	"github.com/adam000/blarg/entry"
	"github.com/adam000/goutils/page"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type DatastoreProvider struct {
	kind         string
	baseUrl      string
	templates    *template.Template
	errorHandler ErrorHandler
}

func NewDatastoreProvider(baseUrl string, kind string, templates *template.Template, errorHandler ErrorHandler) DatastoreProvider {
	return DatastoreProvider{
		kind:         kind,
		baseUrl:      baseUrl,
		templates:    templates,
		errorHandler: errorHandler,
	}
}

func (p DatastoreProvider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Assert the request starts with baseUrl
	requestLower := strings.ToLower(r.URL.String())
	baseLower := strings.ToLower(p.baseUrl)
	if !strings.HasPrefix(requestLower, baseLower) {
		fmt.Fprintf(w, "Fatal error: %s needs to start with %s or else I can't figure out what to do with it", requestLower, baseLower)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}

	// Also remove a dividing slash
	pathRelative := r.URL.String()[len(baseLower)+1:]

	var file entry.File
	client, err := datastore.NewClient(ctx, "adamzerodotnet")
	if err != nil {
		log.Printf("Error making new client: %w", err)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}
	fileKey := datastore.NameKey(p.kind, pathRelative, nil)

	if err := client.Get(ctx, fileKey, &file); err != nil {
		if err == datastore.ErrNoSuchEntity {
			p.errorHandler(w, r, http.StatusNotFound)
			return
		}
		log.Printf("Error getting value (%s | %s): %w", p.kind, pathRelative, err)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}

	p.serveMd(w, r, file.Content)
}

func (p DatastoreProvider) serveMd(w http.ResponseWriter, r *http.Request, markdown string) {
	htmlResult := template.HTML(bluemonday.UGCPolicy().SanitizeBytes(blackfriday.MarkdownCommon([]byte(markdown))))

	// TODO some of this is specific to my website. Abstract it out.
	var page = page.NewPage()
	page.SetTitle("Blarg")
	page.SetSiteTitle("adam0.net")
	page.AddCssFiles(
		"/static/css/base.css",
		"/static/css/header.css",
		"/static/css/blarg-file.css",
	)
	page.AddVar("Content", htmlResult)
	// TODO
	page.AddVar("FileName", "")
	p.templates.ExecuteTemplate(w, "page_blarg_file.html", page)
}
