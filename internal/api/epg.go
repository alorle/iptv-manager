package api

import (
	"context"

	"github.com/alorle/iptv-manager/internal/epg"
)

func (s server) ListEPGChannels(ctx context.Context, request ListEPGChannelsRequestObject) (ListEPGChannelsResponseObject, error) {
	var channels []epg.EPGChannel

	// Check if EPG is available
	if !s.epgUseCase.IsEPGAvailable() {
		code := 500
		msg := "EPG data is not available. Please configure EPG_URL environment variable."
		return ListEPGChannels500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	// If search query is provided, use search; otherwise get all channels
	if request.Params.Search != nil && *request.Params.Search != "" {
		channels = s.epgUseCase.SearchChannels(*request.Params.Search)
	} else {
		channels = s.epgUseCase.ListChannels()
	}

	// Convert domain EPG channels to API EPG channels
	response := ListEPGChannels200JSONResponse{}
	for _, ch := range channels {
		apiChannel := EPGChannel{
			Id:   ch.ID,
			Name: ch.Name,
		}
		if ch.Logo != "" {
			apiChannel.Logo = &ch.Logo
		}
		response = append(response, apiChannel)
	}

	return response, nil
}
