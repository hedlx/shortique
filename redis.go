package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RequestData struct {
	Origin    string
	VideoID   string
	ChatID    int64
	UserID    int64
	MessageID int
	Type      int
}

func (d RequestData) ToRedis() (string, string) {
	return fmt.Sprintf("%d", d.UserID), fmt.Sprintf("%s:%s:%d:%d:%d", d.Origin, d.VideoID, d.ChatID, d.MessageID, d.Type)
}

const (
	originPt    = 0
	videoIDPt   = iota
	chatIDPt    = iota
	messageIDPt = iota
	typePt      = iota
)

func NewData(key, value string) (*RequestData, error) {
	vals := strings.Split(value, ":")
	origin := vals[originPt]
	videoID := vals[videoIDPt]
	userID, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return nil, err
	}

	chatID, err := strconv.ParseInt(vals[chatIDPt], 10, 64)
	if err != nil {
		return nil, err
	}

	messageID, err := strconv.Atoi(vals[messageIDPt])
	if err != nil {
		return nil, err
	}

	typ, err := strconv.Atoi(vals[typePt])
	if err != nil {
		return nil, err
	}

	return &RequestData{
		UserID:    userID,
		VideoID:   videoID,
		ChatID:    chatID,
		MessageID: messageID,
		Type:      typ,
		Origin:    origin,
	}, nil
}

var (
	mehCtx               = context.Background()
	rdb    *redis.Client = redis.NewClient(&redis.Options{
		Addr: mustGetEnv("REDIS"),
	})
)

func WaitForRedis() {
	for {
		if err := rdb.Scan(mehCtx, 0, "", 0).Err(); err == nil {
			return
		}

		time.Sleep(time.Second)
	}
}

func AddReq(req *RequestData) error {
	key, value := req.ToRedis()
	return rdb.SAdd(mehCtx, key, value).Err()
}

func DelReq(req *RequestData) error {
	key, value := req.ToRedis()
	return rdb.SRem(mehCtx, key, value).Err()
}

func ListReqs() <-chan *RequestData {
	resC := make(chan *RequestData, 42)

	go func() {
		var cursor uint64

		for {
			var keys []string
			var err error

			keys, cursor, err = rdb.Scan(mehCtx, cursor, "*", 0).Result()
			if err != nil {
				L.Error(
					"Failed to scan redis",
					zap.Error(err),
				)
				close(resC)
				return
			}

			for _, key := range keys {
				vals, err := rdb.SMembers(mehCtx, key).Result()
				if err != nil {
					L.Error(
						"Failed to get redis members",
						zap.Error(err),
						zap.String("key", key),
					)
					continue
				}

				for _, val := range vals {
					if data, err := NewData(key, val); err != nil {
						L.Error(
							"Failed to parse redis value",
							zap.Error(err),
							zap.String("key", key),
							zap.String("value", val),
						)
					} else {
						resC <- data
					}
				}
			}

			if cursor == 0 {
				close(resC)
				return
			}
		}
	}()

	return resC
}
