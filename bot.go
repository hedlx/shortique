package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type request struct {
	ChatID    int64
	UserID    int64
	MessageID int
	VideoID   string
	Ray       string
}

type result struct {
	req request
	err error
}

func ProxyVideo(bot *tg.BotAPI, req request, done chan<- result) {
	url := MakeYoutubeURL(req.VideoID)
	metadata, err := GetMetadata(url)
	if err != nil {
		done <- result{
			req: req,
			err: err,
		}

		return
	}

	videoReader, waitC, downloadC := DownloadVideoRoutine(url)
	video := tg.NewVideo(req.ChatID, &tg.FileReader{
		Name:   req.VideoID,
		Reader: videoReader,
	})
	video.Caption = metadata.Title
	video.ReplyToMessageID = req.MessageID

	_, sendErr := bot.Send(video)
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
	for {
		select {
		case update := <-updatesC:
			if update.Message == nil {
				break
			}

			for _, videoID := range GetUniqShorts(update.Message.Text) {
				ray := uuid.NewV4().String()
				L.Info(
					"Handle new request",
					zap.String("ray", ray),
					zap.String("video_id", videoID),
					zap.String("user_name", update.Message.From.UserName),
					zap.Int64("user_id", update.Message.From.ID),
					zap.Int64("chat_id", update.Message.Chat.ID),
				)
				go ProxyVideo(
					bot,
					request{
						ChatID:    update.Message.Chat.ID,
						UserID:    update.SentFrom().ID,
						MessageID: update.Message.MessageID,
						VideoID:   videoID,
						Ray:       ray,
					},
					resultsC,
				)
			}
		case result := <-resultsC:
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
