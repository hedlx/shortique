package main

import (
	"encoding/json"
	"regexp"
)

var linkRe = regexp.MustCompile(`https?://((?P<src>www|music)\.)?youtube\.com/(shorts/|watch\?v=)(?P<id>[\w_-]+)`)

const (
	VideoType = iota
	MusicType = iota
)

type Source struct {
	Id   string
	Type int
}

func GetUniqSources(str string) []Source {
	links := linkRe.FindAllString(str, -1)
	sources := make([]Source, 0, len(links))

r:
	for _, link := range links {
		matches := linkRe.FindStringSubmatch(link)
		id := matches[linkRe.SubexpIndex("id")]

		for _, src := range sources {
			if src.Id == id {
				continue r
			}
		}

		src := matches[linkRe.SubexpIndex("src")]
		typ := VideoType
		if src == "music" {
			typ = MusicType
		}

		sources = append(sources, Source{id, typ})
	}

	return sources
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
