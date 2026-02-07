package logging

import (
	"encoding/json"
	"net/http"
)

// HTTPErrorResponse represents a standard JSON error response
type HTTPErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSONError writes a JSON error response and logs it
func WriteJSONError(w http.ResponseWriter, logger *Logger, message string, statusCode int, context map[string]interface{}) {
	// Log error with context
	logFields := make(map[string]interface{})
	if context != nil {
		for k, v := range context {
			logFields[k] = v
		}
	}
	logFields["status_code"] = statusCode
	logFields["message"] = message

	logger.Error("HTTP error response", logFields)

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(HTTPErrorResponse{Error: message}); err != nil {
		logger.Warn("Failed to encode error response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// WriteJSONSuccess writes a JSON success response
func WriteJSONSuccess(w http.ResponseWriter, logger *Logger, data interface{}, context map[string]interface{}) {
	// Log success with optional context
	if logger != nil && context != nil {
		logger.Debug("HTTP success response", context)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		if logger != nil {
			logger.Warn("Failed to encode success response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
