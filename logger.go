package main

import (
	zap "go.uber.org/zap"
)

var L *zap.Logger

func init() {
	var err error

	L, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}
