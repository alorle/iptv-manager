package m3u

import (
	"fmt"
	"io"
	"strings"
)

type encoder struct {
	epgUrls []string
	items   []*Channel
}

func NewEncoder(guideUrls []string) *encoder {
	return &encoder{epgUrls: guideUrls, items: []*Channel{}}
}

func (p *encoder) AddChannel(item *Channel) {
	p.items = append(p.items, item)
}

func (p *encoder) Encode(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "#EXTM3U"); err != nil {
		return err
	}

	if len(p.epgUrls) > 0 {
		if _, err := fmt.Fprintf(w, " tvg-url=\"%s\"", strings.Join(p.epgUrls, ",")); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "#EXTVLCOPT:network-caching=1000\n"); err != nil {
		return err
	}

	for _, item := range p.items {
		if err := item.encode(w); err != nil {
			return err
		}
	}

	return nil
}
