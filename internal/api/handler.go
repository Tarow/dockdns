package api

import (
	"context"
	"net/http"

	template "github.com/Tarow/dockdns/templates"
)

func GetIndex(w http.ResponseWriter, r *http.Request) {

	indexTemplate := template.Index()
	w.WriteHeader(http.StatusOK)
	indexTemplate.Render(context.Background(), w)
}
