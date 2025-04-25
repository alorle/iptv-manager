package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/alorle/iptv-manager/internal/api"
	"github.com/getkin/kin-openapi/openapi3"
)

func NewDocumentationHandler(swagger *openapi3.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := api.GetSwagger()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	})
}
