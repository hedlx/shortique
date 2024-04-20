package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

var linksRe = []string{
	`((?P<src>www|music)\.)?(?P<origin>youtube)\.com/(shorts/|watch\?v=)(?P<id>[\w_-]+)`,
	`(www\.)?(?P<origin>instagram)\.com/reels?/(?P<id>[\w_-]+)`,
}

var prepLinksRe = lo.Map(linksRe, func(l string, _ int) string {
	return "(" + l + ")"
})

var linkRe = regexp.MustCompile("https?://(" + strings.Join(prepLinksRe, "|") + ")")

const (
	VideoType = iota
	MusicType = iota
)

type Source struct {
	Origin string
	Id     string
	Type   int
}

func GetUniqSources(str string) []Source {
	links := linkRe.FindAllString(str, -1)
	sources := make([]Source, 0, len(links))
	groups := linkRe.SubexpNames()
	uniq := map[string]bool{}

	for _, link := range links {
		L.Info("link", zap.String("link", link))
		matches := linkRe.FindStringSubmatch(link)
		source := &Source{
			Type: VideoType,
		}

		for i, match := range matches {
			if match == "" {
				continue
			}

			switch groups[i] {
			case "id":
				source.Id = match
			case "origin":
				source.Origin = match
			case "src":
				if match == "music" {
					source.Type = MusicType
				}
			default:
			}
		}

		L.Debug("result source", zap.Any("source", source))

		if source.Origin == "" || source.Id == "" {
			continue
		}

		key := source.Origin + "_" + source.Id
		if uniq[key] {
			continue
		}

		uniq[key] = true

		sources = append(sources, *source)
	}

	return sources
}

type urlBuilder func(string) string

var urlBuilders = map[string]urlBuilder{
	"youtube": func(id string) string {
		return "https://www.youtube.com/watch?v=" + id
	},
	"instagram": func(id string) string {
		return "https://www.instagram.com/reel/" + id
	},
}

func BuildURL(origin, id string) (string, error) {
	builder := urlBuilders[origin]
	if builder == nil {
		L.Error("Failed to get url builder", zap.String("origin", id))
		return "", fmt.Errorf("no builder for origin: %s", origin)
	}

	return builder(id), nil
}

type VideoMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Channel     string `json:"channel"`
	Artist      string `json:"artist"`
	Track       string `json:"track"`
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
