//go:build !ent

package vercheck

import (
	"net/http"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
)

func addHeaders(req *http.Request) {
	req.Header.Set("User-Agent", cloudapi.UserAgent())
}
