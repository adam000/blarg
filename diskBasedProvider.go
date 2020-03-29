package blarg

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adam000/blarg/rule"
)

type DiskBasedProvider struct {
	baseUrl       string
	baseDirectory string
	rules         []rule.Rule
}

func NewDiskBasedProvider(baseDirectory string, baseUrl string) DiskBasedProvider {
	return DiskBasedProvider{
		baseUrl:       baseUrl,
		baseDirectory: baseDirectory,
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
		return
	}

	// TODO: report if file exists
	pathRelative := r.URL.String()[len(baseLower):]

	pathAbsolute, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), p.baseDirectory, pathRelative))
	if err != nil {
		log.Printf("Error trying to get absolute path of %s: %v", pathRelative, err)
		return
	}

	log.Printf("Looking for file at '%s' (absolute path: '%s')", pathRelative, pathAbsolute)
	if exists, _ := Exists(pathAbsolute); exists {
		log.Printf("File exists!")
	}

	fmt.Fprintf(w, "Done!")
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
