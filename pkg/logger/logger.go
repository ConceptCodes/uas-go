package logger

import (
	"context"
	"os"
	"sync"
	"time"
	"uas/internal/constants"

	"github.com/rs/zerolog"
)

var (
	once sync.Once
	log  *zerolog.Logger
)

func NewWithCtx(ctx context.Context) *zerolog.Logger {
	once.Do(func() {
		output := zerolog.ConsoleWriter{
			Out: os.Stdout,
			FormatTimestamp: func(i interface{}) string {
				parse, _ := time.Parse(time.RFC3339, i.(string))
				return parse.Format(constants.TimeFormat)
			},
		}
		logger := zerolog.New(output).With().Timestamp().Ctx(ctx).CallerWithSkipFrameCount(2).Logger()
		log = &logger
	})
	return log
}

func New() *zerolog.Logger {
	once.Do(func() {
		output := zerolog.ConsoleWriter{
			Out: os.Stdout,
			FormatTimestamp: func(i interface{}) string {
				parse, _ := time.Parse(time.RFC3339, i.(string))
				return parse.Format(constants.TimeFormat)
			},
		}
		logger := zerolog.New(output).With().Timestamp().CallerWithSkipFrameCount(2).Logger()
		log = &logger
	})
	return log
}
