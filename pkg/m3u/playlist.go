package m3u

import (
	"fmt"
	"io"
)

type Playlist struct {
	items []*PlaylistItem
}

func NewPlaylist() *Playlist {
	return &Playlist{items: []*PlaylistItem{}}
}

func (p *Playlist) AppendItem(item *PlaylistItem) {
	p.items = append(p.items, item)
}

func (p *Playlist) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "#EXTM3U\n"); err != nil {
		return err
	}

	for _, item := range p.items {
		if err := item.Write(w); err != nil {
			return err
		}
	}

	return nil
}

type PlaylistItem struct {
	SeqId    uint64
	Title    string
	URI      string
	Duration float64
	TVGTags  *TVGTags
}

func (pi *PlaylistItem) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "#EXTINF:%0.0f", pi.Duration); err != nil {
		return err
	}

	if pi.TVGTags != nil {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}

		if err := pi.TVGTags.Write(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, ",%s\n%s\n", pi.Title, pi.URI); err != nil {
		return err
	}

	return nil
}

type TVGTags struct {
	ID         string
	Name       string
	GroupTitle string
}

func (t *TVGTags) Write(w io.Writer) error {
	if t.ID != "" {
		if _, err := fmt.Fprintf(w, "TVG-ID=\"%s\" ", t.ID); err != nil {
			return err
		}
	}

	if t.Name != "" {
		if _, err := fmt.Fprintf(w, "TVG-NAME=\"%s\" ", t.Name); err != nil {
			return err
		}
	}

	if t.GroupTitle != "" {
		if _, err := fmt.Fprintf(w, "GROUP-TITLE=\"%s\"", t.GroupTitle); err != nil {
			return err
		}
	}

	return nil
}
