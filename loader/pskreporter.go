package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type Report struct {
	SequenceNumber          int    `json:"sequenceNumber"`
	Frequency               int    `json:"frequency"`
	Mode                    string `json:"mode"`
	SNR                     int    `json:"sNR"`
	FlowStartSeconds        int    `json:"flowStartSeconds"`
	SenderCallsign          string `json:"senderCallsign"`
	SenderLocator           string `json:"senderLocator"`
	ReceiverCallsign        string `json:"receiverCallsign"`
	ReceiverLocator         string `json:"receiverLocator"`
	ReceiverDecoderSoftware string `json:"receiverDecoderSoftware"`
	Band                    string `json:"band"`
	SenderCountryName       string `json:"senderCountryName"`
	ReceiverCountryName     string `json:"receiverCountryName"`
	SenderAdifCC            int    `json:"senderAdifCC"`
	ReceiverAdifCC          int    `json:"receiverAdifCC"`
}

func processStream(ctx context.Context, cancel context.CancelCauseFunc, reports chan<- *Report) {
	client := &http.Client{}

	for {
		url := "https://stream.pskreporter.info/stream/report?token=" + url.QueryEscape(config.PSKReporterToken)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			cancel(fmt.Errorf("building request: %w", err))
			return
		}
		res, err := client.Do(req)
		if err != nil {
			cancel(fmt.Errorf("executing pskreporter request: %w", err))
			return
		}
		stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(StatusCode, strconv.Itoa(res.StatusCode))},
			PskReporterStatus.M(1))

		if res.StatusCode != 200 {
			err := fmt.Errorf("got status code %d from pskreporter", res.StatusCode)
			cancel(err)
			return
		}
		lines := bufio.NewScanner(res.Body)
		for lines.Scan() {
			report := &Report{}
			err := json.Unmarshal([]byte(lines.Text()), report)
			stats.Record(ctx, PskReporterLinesRead.M(1))
			if err != nil {
				log.Warn().Err(err).Msg("decoding report")
				stats.Record(ctx, PskReporterInvalidLines.M(1))
				continue
			}
			reports <- report
		}
		log.Warn().Msg("connection dropped, reconnecting")
	}
}
