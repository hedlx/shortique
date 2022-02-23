package main

import (
	"encoding/json"
	"regexp"
)

var shortRe = regexp.MustCompile(`https?://(www\.)?youtube\.com/shorts/(?P<id>[\w_-]+)`)

func GetUniqShorts(str string) []string {
	links := shortRe.FindAllString(str, -1)
	ids := make([]string, 0, len(links))

r:
	for _, link := range links {
		matches := shortRe.FindStringSubmatch(link)
		id := matches[shortRe.SubexpIndex("id")]

		for _, eid := range ids {
			if eid == id {
				continue r
			}
		}

		ids = append(ids, id)
	}

	return ids
}

func MakeYoutubeURL(id string) string {
	return "https://www.youtube.com/watch?v=" + id
}

type VideoMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Channel     string `json:"channel"`
	URL         string `json:"webpage_url"`
	FileName    string `json:"_filename"`
}

func ParseMeta(raw []byte) (*VideoMeta, error) {
	res := &VideoMeta{}
	if err := json.Unmarshal(raw, res); err != nil {
		return nil, err
	}

	return res, nil
}
