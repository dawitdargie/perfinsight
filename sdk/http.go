package sdk

import (
	"fmt"
	"net/http"
)

func HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("middleware reached:", r.URL.Path)
		next(w, r)
	}
}
