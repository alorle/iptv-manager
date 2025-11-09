package api

import (
	"context"

	domain "github.com/alorle/iptv-manager/internal"
)

// Read-only channel view endpoints

func (s server) ListChannels(ctx context.Context, request ListChannelsRequestObject) (ListChannelsResponseObject, error) {
	channels, err := s.streamUseCase.GetChannelViews()
	if err != nil {
		code := 500
		msg := err.Error()
		return ListChannels500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	response := ListChannels200JSONResponse{}
	for _, channel := range channels {
		response = append(response, domainChannelToAPIChannel(channel))
	}
	return response, nil
}

func (s server) GetChannelByGuideId(ctx context.Context, request GetChannelByGuideIdRequestObject) (GetChannelByGuideIdResponseObject, error) {
	channel, err := s.streamUseCase.GetChannelViewByGuideID(request.GuideId)
	if err != nil {
		code := 500
		msg := err.Error()
		return GetChannelByGuideId500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	if channel == nil {
		code := 404
		msg := "Channel not found (no streams with this guide_id)"
		return GetChannelByGuideId404JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return GetChannelByGuideId200JSONResponse(domainChannelToAPIChannel(channel)), nil
}

// Helper functions for conversion

func domainChannelToAPIChannel(channel *domain.Channel) Channel {
	streams := make([]Stream, len(channel.Streams))
	for i, stream := range channel.Streams {
		streamID := stream.ID
		streams[i] = Stream{
			Id:             &streamID,
			GuideId:        stream.GuideID,
			AcestreamId:    stream.AcestreamID,
			Quality:        &stream.Quality,
			Tags:           &stream.Tags,
			NetworkCaching: int(stream.NetworkCaching),
		}
	}

	return Channel{
		GuideId:    channel.GuideID,
		Title:      &channel.Title,
		Logo:       &channel.Logo,
		GroupTitle: &channel.GroupTitle,
		Streams:    &streams,
	}
}
