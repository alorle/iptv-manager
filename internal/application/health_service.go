package application

import (
	"context"

	"github.com/alorle/iptv-manager/internal/port/driven"
)

// HealthService orchestrates health checks for the application and its dependencies.
type HealthService struct {
	db     driven.ChannelRepository
	engine driven.AceStreamEngine
}

// NewHealthService creates a new health check service.
func NewHealthService(db driven.ChannelRepository, engine driven.AceStreamEngine) *HealthService {
	return &HealthService{
		db:     db,
		engine: engine,
	}
}

// ComponentHealth represents the health status of a single component.
type ComponentHealth struct {
	Status string // "ok" or "error"
	Error  string // empty if status is "ok", otherwise contains error message
}

// HealthStatus represents the overall health status of the application.
type HealthStatus struct {
	Status          string          // "ok" if all components are healthy, "degraded" otherwise
	DB              ComponentHealth // database health
	AceStreamEngine ComponentHealth // acestream engine health
}

// Check performs health checks on all dependencies.
// Returns the overall health status and individual component statuses.
func (s *HealthService) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status: "ok",
	}

	// Check database health
	if err := s.db.Ping(ctx); err != nil {
		status.DB = ComponentHealth{
			Status: "error",
			Error:  err.Error(),
		}
		status.Status = "degraded"
	} else {
		status.DB = ComponentHealth{
			Status: "ok",
		}
	}

	// Check AceStream engine health
	if err := s.engine.Ping(ctx); err != nil {
		status.AceStreamEngine = ComponentHealth{
			Status: "error",
			Error:  err.Error(),
		}
		status.Status = "degraded"
	} else {
		status.AceStreamEngine = ComponentHealth{
			Status: "ok",
		}
	}

	return status
}
