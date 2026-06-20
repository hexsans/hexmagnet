package testutil

import "net/http"

func ESOK(w http.ResponseWriter, body string) {
	w.Header().Set("X-Elastic-Product", "Elasticsearch")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
