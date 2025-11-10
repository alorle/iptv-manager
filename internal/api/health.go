package api

import (
	"context"
	"time"
)

// GetHealth implements the health check endpoint
func (s *Server) GetHealth(_ context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
	}, nil
}
