package application

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/probe"
	"github.com/alorle/iptv-manager/internal/stream"
)

// PlaylistService provides use cases for playlist generation.
// It depends only on port interfaces.
type PlaylistService struct {
	streamRepo driven.StreamRepository
	probeRepo  driven.ProbeRepository
	window     time.Duration
}

// NewPlaylistService creates a new PlaylistService with the given dependencies.
func NewPlaylistService(
	streamRepo driven.StreamRepository,
	probeRepo driven.ProbeRepository,
	window time.Duration,
) *PlaylistService {
	return &PlaylistService{
		streamRepo: streamRepo,
		probeRepo:  probeRepo,
		window:     window,
	}
}

// GenerateM3U generates an M3U playlist with all available streams.
// The host parameter is used to build the proxy URL for each stream.
// Returns a playlist with only the #EXTM3U header if no streams are found.
func (p *PlaylistService) GenerateM3U(ctx context.Context, host string) (string, error) {
	streams, err := p.streamRepo.FindAll(ctx)
	if err != nil {
		return "", err
	}

	sorted := p.sortByQuality(ctx, streams)

	var builder strings.Builder
	builder.WriteString("#EXTM3U\n")

	for _, s := range sorted {
		// Format: #EXTINF:-1 tvg-id="NombreCanal",NombreCanal - infohash
		fmt.Fprintf(&builder, "#EXTINF:-1 tvg-id=\"%s\",%s - %s\n",
			s.ChannelName(),
			s.ChannelName(),
			s.InfoHash())

		// Format: http://{host}/ace/getstream?id=infohash
		fmt.Fprintf(&builder, "http://%s/ace/getstream?id=%s\n",
			host,
			s.InfoHash())
	}

	return builder.String(), nil
}

// sortByQuality groups streams by channel name, sorts channel groups
// alphabetically, and within each group sorts streams by quality score
// descending. Streams without probe data sort after scored streams,
// with infohash as the final tiebreaker.
func (p *PlaylistService) sortByQuality(ctx context.Context, streams []stream.Stream) []stream.Stream {
	groups := make(map[string][]stream.Stream)
	var channelNames []string
	for _, s := range streams {
		name := s.ChannelName()
		if _, exists := groups[name]; !exists {
			channelNames = append(channelNames, name)
		}
		groups[name] = append(groups[name], s)
	}

	slices.Sort(channelNames)

	since := time.Now().Add(-p.window)

	var result []stream.Stream
	for _, name := range channelNames {
		result = append(result, p.sortGroupByQuality(ctx, groups[name], since)...)
	}

	return result
}

type scoredStream struct {
	s        stream.Stream
	score    float64
	hasScore bool
}

// sortGroupByQuality sorts streams within a single channel group by
// quality score descending, using per-group normalization ceilings.
func (p *PlaylistService) sortGroupByQuality(ctx context.Context, group []stream.Stream, since time.Time) []stream.Stream {
	metricsMap := make(map[string]probe.Metrics, len(group))
	for _, s := range group {
		results, err := p.probeRepo.FindByInfoHashSince(ctx, s.InfoHash(), since)
		if err != nil {
			continue
		}
		m, err := probe.NewMetrics(s.InfoHash(), results)
		if err != nil {
			continue
		}
		metricsMap[s.InfoHash()] = m
	}

	var maxSpeed, maxPeers float64
	for _, m := range metricsMap {
		if m.AvgDownloadSpeed() > maxSpeed {
			maxSpeed = m.AvgDownloadSpeed()
		}
		if m.AvgPeerCount() > maxPeers {
			maxPeers = m.AvgPeerCount()
		}
	}

	scored := make([]scoredStream, 0, len(group))
	for _, s := range group {
		m, ok := metricsMap[s.InfoHash()]
		if ok {
			score := probe.ComputeQualityScore(m, maxSpeed, maxPeers)
			scored = append(scored, scoredStream{s: s, score: score, hasScore: true})
		} else {
			scored = append(scored, scoredStream{s: s, hasScore: false})
		}
	}

	slices.SortFunc(scored, func(a, b scoredStream) int {
		if a.hasScore != b.hasScore {
			if a.hasScore {
				return -1
			}
			return 1
		}
		if a.hasScore {
			if c := cmp.Compare(b.score, a.score); c != 0 {
				return c
			}
		}
		return cmp.Compare(a.s.InfoHash(), b.s.InfoHash())
	})

	result := make([]stream.Stream, len(scored))
	for i, ss := range scored {
		result[i] = ss.s
	}
	return result
}
