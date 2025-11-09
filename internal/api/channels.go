package api

import (
	"context"
	"errors"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/alorle/iptv-manager/internal/memory"
	"github.com/google/uuid"
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
		response = append(response, domainChannelToAPIChannel(channel))
	}
	return response, nil
}

func (s server) GetChannel(ctx context.Context, request GetChannelRequestObject) (GetChannelResponseObject, error) {
	channel, err := s.getChannelsUseCase.GetChannel(request.Id)
	if err != nil {
		if errors.Is(err, memory.ErrChannelNotFound) {
			code := 404
			msg := "Channel not found"
			return GetChannel404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return GetChannel500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return GetChannel200JSONResponse(domainChannelToAPIChannel(channel)), nil
}

func (s server) CreateChannel(ctx context.Context, request CreateChannelRequestObject) (CreateChannelResponseObject, error) {
	channel := apiChannelToDomainChannel((*Channel)(request.Body))

	if err := s.getChannelsUseCase.CreateChannel(channel); err != nil {
		if errors.Is(err, memory.ErrChannelExists) {
			code := 400
			msg := "Channel with this ID already exists"
			return CreateChannel400JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return CreateChannel500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return CreateChannel201JSONResponse(domainChannelToAPIChannel(channel)), nil
}

func (s server) UpdateChannel(ctx context.Context, request UpdateChannelRequestObject) (UpdateChannelResponseObject, error) {
	channel := apiChannelToDomainChannel((*Channel)(request.Body))

	if err := s.getChannelsUseCase.UpdateChannel(request.Id, channel); err != nil {
		if errors.Is(err, memory.ErrChannelNotFound) {
			code := 404
			msg := "Channel not found"
			return UpdateChannel404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return UpdateChannel500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return UpdateChannel200JSONResponse(domainChannelToAPIChannel(channel)), nil
}

func (s server) DeleteChannel(ctx context.Context, request DeleteChannelRequestObject) (DeleteChannelResponseObject, error) {
	if err := s.getChannelsUseCase.DeleteChannel(request.Id); err != nil {
		if errors.Is(err, memory.ErrChannelNotFound) {
			code := 404
			msg := "Channel not found"
			return DeleteChannel404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return DeleteChannel500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return DeleteChannel204Response{}, nil
}

// Helper functions for conversion

func domainChannelToAPIChannel(channel *domain.Channel) Channel {
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

	return Channel{
		Id:         channel.ID,
		Title:      channel.Title,
		GuideId:    channel.GuideID,
		Logo:       &channel.Logo,
		GroupTitle: channel.GroupTitle,
		Streams:    streams,
	}
}

func apiChannelToDomainChannel(channel *Channel) *domain.Channel {
	streams := make([]*domain.Stream, len(channel.Streams))
	for i, stream := range channel.Streams {
		quality := ""
		if stream.Quality != nil {
			quality = *stream.Quality
		}
		tags := []string{}
		if stream.Tags != nil {
			tags = *stream.Tags
		}

		streams[i] = &domain.Stream{
			ID:             stream.Id,
			ChannelID:      stream.ChannelId,
			AcestreamID:    stream.AcestreamId,
			Quality:        quality,
			Tags:           tags,
			NetworkCaching: uint64(stream.NetworkCaching),
		}
	}

	logo := ""
	if channel.Logo != nil {
		logo = *channel.Logo
	}

	return &domain.Channel{
		ID:         channel.Id,
		Title:      channel.Title,
		GuideID:    channel.GuideId,
		Logo:       logo,
		GroupTitle: channel.GroupTitle,
		Streams:    streams,
	}
}

// UUID helper function for consistency
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
