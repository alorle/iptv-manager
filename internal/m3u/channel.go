package m3u

import (
	"fmt"
	"io"
)

type Channel struct {
	SeqId    uint64
	Title    string
	URI      string
	Duration float64
	TVGTags  *TVGTags
}

func (pi *Channel) encode(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "#EXTINF:%0.0f", pi.Duration); err != nil {
		return err
	}

	if pi.TVGTags != nil {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}

		if err := pi.TVGTags.encode(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, ",%s\n%s\n", pi.Title, pi.URI); err != nil {
		return err
	}

	return nil
}
