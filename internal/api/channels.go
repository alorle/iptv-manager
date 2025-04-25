package api

import (
	"context"
)

func (s server) ListChannels(ctx context.Context, request ListChannelsRequestObject) (ListChannelsResponseObject, error) {
	channels, err := s.getChannelsUseCase.GetChannels()
	if err != nil {
		code := 500
		msg := err.Error()
		return ListChannels500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	response := ListChannels200JSONResponse{}
	for _, channel := range channels {
		response = append(response, Channel{
			Id:          channel.ID,
			Name:        channel.Title,
			AcestreamId: channel.StreamID,
			Category:    &channel.GroupTitle,
			EpgId:       &channel.GuideID,
			Quality:     &channel.Quality,
			Tags:        &channel.Tags,
			CreatedAt:   nil,
			UpdatedAt:   nil,
		})
	}
	return response, nil
}
