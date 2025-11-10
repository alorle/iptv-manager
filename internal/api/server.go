package api

//go:generate go tool oapi-codegen -config cfg.yaml ../../openapi.yaml

// Server implements the API endpoints
type Server struct{}

// NewServer creates a new API server instance
func NewServer() *Server {
	return &Server{}
}

// ensure that we've conformed to the `StrictServerInterface` with a compile-time check
var _ StrictServerInterface = (*Server)(nil)
