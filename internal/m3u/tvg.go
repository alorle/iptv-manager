package m3u

import (
	"fmt"
	"io"
)

type TVGTags struct {
	ID         string
	Name       string
	GroupTitle string
}

func (t *TVGTags) encode(w io.Writer) error {
	if t.ID != "" {
		if _, err := fmt.Fprintf(w, "tvg-id=\"%s\" ", t.ID); err != nil {
			return err
		}
	}

	if t.Name != "" {
		if _, err := fmt.Fprintf(w, "tvg-name=\"%s\" ", t.Name); err != nil {
			return err
		}
	}

	if t.GroupTitle != "" {
		if _, err := fmt.Fprintf(w, "group-title=\"%s\"", t.GroupTitle); err != nil {
			return err
		}
	}

	return nil
}
