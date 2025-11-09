package api

import "github.com/alorle/iptv-manager/internal/usecase"

//go:generate go tool oapi-codegen -config cfg.yaml ../../openapi.yaml

// ensure that we've conformed to the `StrictServerInterface ` with a compile-time check
var _ StrictServerInterface = (*server)(nil)

type server struct {
	getChannelsUseCase usecase.GetChannelUseCase
	epgUseCase         *usecase.EPGUseCase
}

func NewServer(
	getChannelsUseCase usecase.GetChannelUseCase,
	epgUseCase *usecase.EPGUseCase,
) server {
	return server{
		getChannelsUseCase: getChannelsUseCase,
		epgUseCase:         epgUseCase,
	}
}
