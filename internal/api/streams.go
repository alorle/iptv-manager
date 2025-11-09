package api

import (
	"context"
	"errors"

	domain "github.com/alorle/iptv-manager/internal"
	"github.com/alorle/iptv-manager/internal/memory"
	"github.com/google/uuid"
)

func (s server) ListStreams(ctx context.Context, request ListStreamsRequestObject) (ListStreamsResponseObject, error) {
	streams, err := s.streamUseCase.GetStreams()
	if err != nil {
		code := 500
		msg := err.Error()
		return ListStreams500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	response := ListStreams200JSONResponse{}
	for _, stream := range streams {
		response = append(response, domainStreamToAPIStream(stream))
	}
	return response, nil
}

func (s server) GetStream(ctx context.Context, request GetStreamRequestObject) (GetStreamResponseObject, error) {
	stream, err := s.streamUseCase.GetStream(request.Id)
	if err != nil {
		if errors.Is(err, memory.ErrStreamNotFound) {
			code := 404
			msg := "Stream not found"
			return GetStream404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return GetStream500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return GetStream200JSONResponse(domainStreamToAPIStream(stream)), nil
}

func (s server) CreateStream(ctx context.Context, request CreateStreamRequestObject) (CreateStreamResponseObject, error) {
	stream := apiStreamToDomainStream((*Stream)(request.Body))

	// Validate guide_id against EPG if EPG is available
	if s.epgUseCase.IsEPGAvailable() && !s.epgUseCase.ValidateGuideID(stream.GuideID) {
		code := 400
		msg := "Invalid guide_id: channel not found in EPG. Please select a channel from the EPG list."
		return CreateStream400JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	if err := s.streamUseCase.CreateStream(stream); err != nil {
		if errors.Is(err, memory.ErrStreamExists) {
			code := 400
			msg := "Stream with this ID already exists"
			return CreateStream400JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return CreateStream500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return CreateStream201JSONResponse(domainStreamToAPIStream(stream)), nil
}

func (s server) UpdateStream(ctx context.Context, request UpdateStreamRequestObject) (UpdateStreamResponseObject, error) {
	stream := apiStreamToDomainStream((*Stream)(request.Body))

	// Validate guide_id against EPG if EPG is available
	if s.epgUseCase.IsEPGAvailable() && !s.epgUseCase.ValidateGuideID(stream.GuideID) {
		code := 400
		msg := "Invalid guide_id: channel not found in EPG. Please select a channel from the EPG list."
		return UpdateStream400JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	if err := s.streamUseCase.UpdateStream(request.Id, stream); err != nil {
		if errors.Is(err, memory.ErrStreamNotFound) {
			code := 404
			msg := "Stream not found"
			return UpdateStream404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return UpdateStream500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return UpdateStream200JSONResponse(domainStreamToAPIStream(stream)), nil
}

func (s server) DeleteStream(ctx context.Context, request DeleteStreamRequestObject) (DeleteStreamResponseObject, error) {
	if err := s.streamUseCase.DeleteStream(request.Id); err != nil {
		if errors.Is(err, memory.ErrStreamNotFound) {
			code := 404
			msg := "Stream not found"
			return DeleteStream404JSONResponse(Error{Code: &code, Message: &msg}), nil
		}
		code := 500
		msg := err.Error()
		return DeleteStream500JSONResponse(Error{Code: &code, Message: &msg}), nil
	}

	return DeleteStream204Response{}, nil
}

// Helper functions for conversion

func domainStreamToAPIStream(stream *domain.Stream) Stream {
	streamID := stream.ID
	return Stream{
		Id:             &streamID,
		GuideId:        stream.GuideID,
		AcestreamId:    stream.AcestreamID,
		Quality:        &stream.Quality,
		Tags:           &stream.Tags,
		NetworkCaching: int(stream.NetworkCaching),
	}
}

func apiStreamToDomainStream(stream *Stream) *domain.Stream {
	quality := ""
	if stream.Quality != nil {
		quality = *stream.Quality
	}
	tags := []string{}
	if stream.Tags != nil {
		tags = *stream.Tags
	}

	var streamID uuid.UUID
	if stream.Id != nil {
		streamID = uuid.UUID(*stream.Id)
	}
	return &domain.Stream{
		ID:             streamID,
		GuideID:        stream.GuideId,
		AcestreamID:    stream.AcestreamId,
		Quality:        quality,
		Tags:           tags,
		NetworkCaching: uint64(stream.NetworkCaching),
	}
}
