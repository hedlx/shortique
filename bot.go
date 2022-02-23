package main

import (
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

	videoReader, waitC, downloadC := DownloadVideoRoutine(url)
	video := tg.NewVideo(req.Data.ChatID, &tg.FileReader{
		Name:   req.Data.VideoID,
		Reader: videoReader,
	})
	video.Caption = metadata.Title
	video.ReplyToMessageID = req.Data.MessageID

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

			for _, videoID := range GetUniqShorts(update.Message.Text) {
				handleRequest(&RequestData{
					ChatID:    update.Message.Chat.ID,
					UserID:    update.SentFrom().ID,
					MessageID: update.Message.MessageID,
					VideoID:   videoID,
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
