package blarg

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adam000/blarg/rule"
	"github.com/adam000/goutils/page"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// TODO at some point this type should change -- some statuses are more than just a code
type ErrorHandler func(http.ResponseWriter, *http.Request, int)

type DiskBasedProvider struct {
	baseUrl       string
	baseDirectory string
	rules         []rule.Rule
	templates     *template.Template
	errorHandler  ErrorHandler
}

func NewDiskBasedProvider(baseDirectory string, baseUrl string, templates *template.Template, errorHandler ErrorHandler) DiskBasedProvider {
	return DiskBasedProvider{
		baseUrl:       baseUrl,
		baseDirectory: baseDirectory,
		templates:     templates,
		errorHandler:  errorHandler,
	}
}

func (p *DiskBasedProvider) SetBaseUrl(base string) {
	p.baseUrl = base
}

func (p DiskBasedProvider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Assert the request starts with baseUrl
	requestLower := strings.ToLower(r.URL.String())
	baseLower := strings.ToLower(p.baseUrl)
	if !strings.HasPrefix(requestLower, baseLower) {
		fmt.Fprintf(w, "Fatal error: %s needs to start with %s or else I can't figure out what to do with it", requestLower, baseLower)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}

	pathRelative := r.URL.String()[len(baseLower):]

	pathAbsolute, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), p.baseDirectory, pathRelative))
	if err != nil {
		log.Printf("Error trying to get absolute path of %s: %v", pathRelative, err)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}

	if strings.EqualFold(pathRelative, "/changelog") {
		log.Printf("Serving up changelog")
		p.serveChangelog(w, r)
		return
	}

	log.Printf("Looking for file at '%s' (absolute path: '%s')", pathRelative, pathAbsolute)
	if exists, _ := Exists(pathAbsolute); !exists {
		mdPath := pathAbsolute + ".md"
		if mdExists, _ := Exists(mdPath); mdExists {
			log.Printf("Serving up %s", mdPath)
			p.serveMd(w, r, mdPath)
			return
		}

		log.Printf("File does not exist!")
		p.errorHandler(w, r, http.StatusNotFound)
		return
	}

	if isDir, _ := IsDir(pathAbsolute); isDir {
		fmt.Fprintf(w, "TODO implement directory support")
		return
	}

	http.ServeFile(w, r, pathAbsolute)
}

func (p DiskBasedProvider) serveChangelog(w http.ResponseWriter, r *http.Request) {
	var output bytes.Buffer
	paths := make([]string, 0)
	err := filepath.WalkDir("blarg/changelog", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && d.Name() != "TEMPLATE.md" {
			// Queue the names of files and then print them in reverse so that
			// it prints from oldest to newest.
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil {
		log.Printf("%s", err)
		p.errorHandler(w, r, http.StatusInternalServerError)
		return
	}

	for i := len(paths) - 1; i >= 0; i-- {
		bytes, err := os.ReadFile(paths[i])
		if err != nil {
			log.Printf("%s", err)
			p.errorHandler(w, r, http.StatusInternalServerError)
			return
		}
		output.Write(bytes)
		output.Write([]byte{10, 10})
	}

	htmlResult := template.HTML(bluemonday.UGCPolicy().SanitizeBytes(blackfriday.MarkdownCommon(output.Bytes())))

	// TODO some of this is specific to my website. Abstract it out.
	var page = page.NewPage()
	page.SetTitle("Blarg")
	page.AddCssFiles(
		"/static/css/base.css",
		"/static/css/header.css",
		"/static/css/blarg-file.css",
	)
	page.AddVar("Content", htmlResult)
	page.AddVar("FileName", "Changelog")
	p.templates.ExecuteTemplate(w, "page_blarg_file.html", page)
}

func (p DiskBasedProvider) serveMd(w http.ResponseWriter, r *http.Request, mdPath string) {
	markdown, err := os.ReadFile(mdPath)
	if err != nil {
		log.Printf("Error reading MD file: %s", markdown)
		p.errorHandler(w, r, http.StatusInternalServerError)
	}

	htmlResult := template.HTML(bluemonday.UGCPolicy().SanitizeBytes(blackfriday.MarkdownCommon(markdown)))

	// TODO some of this is specific to my website. Abstract it out.
	var page = page.NewPage()
	page.SetTitle("Blarg")
	page.AddCssFiles(
		"/static/css/base.css",
		"/static/css/header.css",
		"/static/css/blarg-file.css",
	)
	page.AddVar("Content", htmlResult)
	page.AddVar("FileName", strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath)))
	p.templates.ExecuteTemplate(w, "page_blarg_file.html", page)
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist), err
}

func IsDir(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return stat.IsDir(), nil
}
