package api

import (
	"log/slog"
	"net/http"

	"github.com/Tarow/dockdns/internal/dns"
	template "github.com/Tarow/dockdns/templates"
)

type Handler struct {
	dnsHandler *dns.Handler
}

func NewHandler(dnsHandler *dns.Handler) Handler {
	return Handler{
		dnsHandler: dnsHandler,
	}
}

func (h Handler) GetIndex(w http.ResponseWriter, r *http.Request) {
	indexTemplate := template.Index(h.dnsHandler.DnsCfg, h.dnsHandler.LatestDomains, h.dnsHandler.LastUpdate)
	w.WriteHeader(http.StatusOK)
	err := indexTemplate.Render(r.Context(), w)
	if err != nil {
		slog.Warn("failed to render index page", "err", err)
	}
}
