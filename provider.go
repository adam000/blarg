package blarg

import "net/http"
import "net/url"

type Provider interface {
	SetBaseUrl(*url.URL)
	ServeHTTP(http.ResponseWriter, *http.Request)
}
