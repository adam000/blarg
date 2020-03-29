package rule

import "net/http"

type Rule interface {
	IsRoutable(string) bool
	Handle(http.ResponseWriter, *http.Request)
}
