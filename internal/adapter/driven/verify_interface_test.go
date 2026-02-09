package driven

import (
	port "github.com/alorle/iptv-manager/internal/port/driven"
)

// Compile-time check that EPGXMLFetcher implements EPGFetcher interface
var _ port.EPGFetcher = (*EPGXMLFetcher)(nil)

// Compile-time check that SubscriptionBoltDBRepository implements SubscriptionRepository interface
var _ port.SubscriptionRepository = (*SubscriptionBoltDBRepository)(nil)
