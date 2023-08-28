//go:build !ent

package vercheck

import (
	"fmt"
	"net/http"
	"runtime"
)

func addHeaders(req *http.Request) {
	req.Header.Set(
		"User-Agent",
		fmt.Sprintf("Atlas-CLI (%s/%s)", runtime.GOOS, runtime.GOARCH),
	)
}
