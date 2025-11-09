package epg

import (
	"encoding/xml"
	"io"
)

// XMLTV represents the root element of an XMLTV EPG file
type XMLTV struct {
	XMLName  xml.Name       `xml:"tv"`
	Channels []XMLTVChannel `xml:"channel"`
}

// XMLTVChannel represents a channel element in XMLTV
type XMLTVChannel struct {
	ID          string   `xml:"id,attr"`
	DisplayName []string `xml:"display-name"`
	Icon        *struct {
		Src string `xml:"src,attr"`
	} `xml:"icon"`
}

// ParseXMLTV parses XMLTV data from a reader and returns EPG channels
func ParseXMLTV(r io.Reader) ([]EPGChannel, error) {
	var xmltv XMLTV
	decoder := xml.NewDecoder(r)

	if err := decoder.Decode(&xmltv); err != nil {
		return nil, err
	}

	channels := make([]EPGChannel, 0, len(xmltv.Channels))
	for _, ch := range xmltv.Channels {
		name := ch.ID // Default to ID if no display name
		if len(ch.DisplayName) > 0 {
			name = ch.DisplayName[0] // Use first display name
		}

		logo := ""
		if ch.Icon != nil {
			logo = ch.Icon.Src
		}

		channels = append(channels, EPGChannel{
			ID:   ch.ID,
			Name: name,
			Logo: logo,
		})
	}

	return channels, nil
}
