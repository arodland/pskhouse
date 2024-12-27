package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sources/flag"
)

type Config struct {
	ClickhouseAddr     string        `dialsdesc:"ClickHouse database host[:port]"`
	ClickhouseDB       string        `dialsdesc:"ClickHouse database name"`
	ClickhouseUsername string        `dialsdesc:"ClickHouse username"`
	ClickhousePassword string        `dialsdesc:"ClickHouse password"`
	ClickhouseTable    string        `dialsdesc:"Table to insert reports into"`
	PSKReporterToken   string        `dialsdesc:"PSKReporter stream token"`
	FlushFrequency     time.Duration `dialsdesc:"How often to commit data to the DB"`
	MetricsPort        int           `dialsdesc:"Port to listen on for metrics"`
}

func defaultConfig() *Config {
	return &Config{
		ClickhouseAddr:     "127.0.0.1:9000",
		ClickhouseDB:       "pskhouse",
		ClickhouseUsername: "pskhouse",
		ClickhouseTable:    "rx",
		FlushFrequency:     time.Second,
		MetricsPort:        9001,
	}
}

var config *Config

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	log.Logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out: os.Stderr,
		},
	).With().Timestamp().Logger()

	config = defaultConfig()
	flagSrc, err := flag.NewCmdLineSet(flag.DefaultFlagNameConfig(), config)
	if err != nil {
		log.Fatal().Err(err).Msg("creating cmdlineset")
	}
	d, err := dials.Config(ctx, config, flagSrc)
	if err != nil {
		log.Fatal().Err(err).Msg("setting dials config")
	}
	config = d.View()

	initMetrics()
	go metricsServer()

	reports := make(chan *Report, 100)
	go processStream(ctx, cancel, reports)
	go insertData(ctx, cancel, reports)

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			switch err {
			case context.Canceled:
				cause := context.Cause(ctx)
				log.Fatal().Err(cause).Msg("shutting down")
			case nil:
				return
			default:
				log.Fatal().Err(err).Msg("shutting down on context error")
			}
		}
	}
}
