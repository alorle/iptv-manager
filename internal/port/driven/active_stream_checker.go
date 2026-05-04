package driven

// ActiveStreamChecker checks whether a stream is currently being consumed by clients.
type ActiveStreamChecker interface {
	IsStreamActive(infoHash string) bool
}
