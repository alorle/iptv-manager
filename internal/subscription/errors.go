package subscription

import "errors"

// Domain errors for subscription operations.
var (
	// Subscription validation errors
	ErrEmptyEPGChannelID = errors.New("subscription epg channel id cannot be empty")

	// Subscription operation errors
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
)
