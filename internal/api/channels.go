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
		streams := make([]Stream, len(channel.Streams))
		for i, stream := range channel.Streams {
			streams[i] = Stream{
				Id:             stream.ID,
				ChannelId:      stream.ChannelID,
				AcestreamId:    stream.AcestreamID,
				Quality:        &stream.Quality,
				Tags:           &stream.Tags,
				NetworkCaching: int(stream.NetworkCaching),
			}
		}

		response = append(response, Channel{
			Id:         channel.ID,
			Title:      channel.Title,
			GuideId:    channel.GuideID,
			Logo:       &channel.Logo,
			GroupTitle: channel.GroupTitle,
			Streams:    streams,
		})
	}
	return response, nil
}
