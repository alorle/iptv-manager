package api

import (
	"context"
)

// GetConfig implements GET /config
func (s server) GetConfig(ctx context.Context, request GetConfigRequestObject) (GetConfigResponseObject, error) {
	return GetConfig200JSONResponse{
		AcestreamUrl: s.acestreamURL,
	}, nil
}
