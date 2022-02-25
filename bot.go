package main

import (
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type request struct {
	Data *RequestData
	Ray  string
}

type result struct {
	req request
	err error
}

func UniqArtists(artists []string) []string {
	used := map[string]bool{}
	res := []string{}

	for _, a := range artists {
		la := strings.ToLower(a)

		if used[la] {
			continue
		}

		used[la] = true
		res = append(res, a)
	}

	return res
}

func ExtractAudioInfo(meta *VideoMeta) (string, string) {
	artists := strings.Split(meta.Artist, ", ")
	if len(artists) == 0 {
		artists = append(artists, meta.Channel)
	}

	artist := strings.Join(UniqArtists(artists), ", ")
	track := meta.Track

	if len(track) == 0 {
		track = meta.Title
	}
	return artist, track
}

func ProxyVideo(bot *tg.BotAPI, req request, done chan<- result) {
	url := MakeYoutubeURL(req.Data.VideoID)
	metadata, err := GetMetadata(url)
	if err != nil {
		done <- result{
			req: req,
			err: err,
		}

		return
	}

	videoReader, waitC, downloadC := DownloadVideoRoutine(VideoSource{url, req.Data.Type})
	def := tg.NewMessage(req.Data.ChatID, "This should never happen!")
	def.ReplyToMessageID = req.Data.MessageID

	var message tg.Chattable = def

	if req.Data.Type == VideoType {
		actual := tg.NewVideo(req.Data.ChatID, &tg.FileReader{
			Name:   req.Data.VideoID + ".mp4",
			Reader: videoReader,
		})
		actual.Caption = metadata.Title
		actual.ReplyToMessageID = req.Data.MessageID

		message = actual
	} else {
		actual := tg.NewAudio(req.Data.ChatID, &tg.FileReader{
			Name:   req.Data.VideoID + ".mp3",
			Reader: videoReader,
		})
		actual.ReplyToMessageID = req.Data.MessageID
		actual.Performer, actual.Title = ExtractAudioInfo(metadata)

		message = actual
	}

	_, sendErr := bot.Send(message)
	close(waitC)
	err = <-downloadC

	if err == nil {
		err = sendErr
	}

	done <- result{
		req: req,
		err: err,
	}
}

func RunBot(token string) error {
	bot, err := tg.NewBotAPI(token)

	if err != nil {
		return err
	}

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updatesC := bot.GetUpdatesChan(u)
	resultsC := make(chan result)

	handleRequest := func(reqData *RequestData) {
		ray := uuid.NewV4().String()
		req := request{
			Data: reqData,
			Ray:  ray,
		}

		if err := AddReq(reqData); err != nil {
			resultsC <- result{
				req: req,
				err: err,
			}

			return
		}

		L.Info(
			"Handle new request",
			zap.String("ray", ray),
			zap.String("video_id", reqData.VideoID),
			zap.Int64("user_id", reqData.UserID),
			zap.Int64("chat_id", reqData.ChatID),
		)
		go ProxyVideo(bot, req, resultsC)
	}

	for req := range ListReqs() {
		handleRequest(req)
	}

	for {
		select {
		case update := <-updatesC:
			if update.Message == nil {
				break
			}

			for _, src := range GetUniqSources(update.Message.Text) {
				handleRequest(&RequestData{
					ChatID:    update.Message.Chat.ID,
					UserID:    update.SentFrom().ID,
					MessageID: update.Message.MessageID,
					VideoID:   src.Id,
					Type:      src.Type,
				})
			}
		case result := <-resultsC:
			if err := DelReq(result.req.Data); err != nil {
				L.Error(
					"Failed to clean up request",
					zap.String("ray", result.req.Ray),
					zap.Error(result.err),
				)
			}

			if result.err != nil {
				L.Error(
					"Failed to handle request",
					zap.String("ray", result.req.Ray),
					zap.Error(result.err),
				)
			} else {
				L.Info(
					"Finished",
					zap.String("ray", result.req.Ray),
				)
			}
		}
	}
}
