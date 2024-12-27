package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	maidenhead "github.com/pd0mz/go-maidenhead"
	"github.com/rs/zerolog/log"
	"go.opencensus.io/stats"
)

type ClickhouseRow struct {
	ID        uint64    `ch:"id"`
	Time      time.Time `ch:"time"`
	Band      int16     `ch:"band"`
	Frequency uint32    `ch:"frequency"`
	SNR       int8      `ch:"snr"`
	Mode      string    `ch:"mode"`

	Version string `ch:"version"`

	RXSign string  `ch:"rx_sign"`
	RXLat  float32 `ch:"rx_lat"`
	RXLon  float32 `ch:"rx_lon"`
	RXLoc  string  `ch:"rx_loc"`

	TXSign string  `ch:"tx_sign"`
	TXLat  float32 `ch:"tx_lat"`
	TXLon  float32 `ch:"tx_lon"`
	TXLoc  string  `ch:"tx_loc"`

	Distance  uint16 `ch:"distance"`
	Azimuth   uint16 `ch:"azimuth"`
	RXAzimuth uint16 `ch:"rx_azimuth"`
}

func insertData(ctx context.Context, cancel context.CancelCauseFunc, reports <-chan *Report) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{config.ClickhouseAddr},
		Auth: clickhouse.Auth{
			Database: config.ClickhouseDB,
			Username: config.ClickhouseUsername,
			Password: config.ClickhousePassword,
		},
	})
	if err != nil {
		cancel(err)
		return
	}

	insertStmt := "INSERT INTO " + config.ClickhouseTable
	batch, err := conn.PrepareBatch(ctx, insertStmt)
	if err != nil {
		cancel(fmt.Errorf("preparing initial batch: %w", err))
		return
	}

	flushTicker := time.NewTicker(config.FlushFrequency)
	defer flushTicker.Stop()

	for {
		select {
		case report := <-reports:
			row, err := convertRow(report)
			if err != nil {
				log.Warn().Err(err).Interface("report", report).Msg("converting report")
				stats.Record(ctx, ClickHouseConvertErrors.M(1))
				continue
			}
			err = batch.AppendStruct(row)
			if err != nil {
				log.Error().Err(err).Interface("row", row).Msg("appending row to batch")
				continue
			}
		case <-flushTicker.C:
			if rows := batch.Rows(); rows > 0 {
				stats.Record(ctx,
					ClickHouseBatches.M(1),
					ClickHouseRows.M(int64(rows)),
					ClickHouseBatchSize.M(float64(rows)),
				)
				err = batch.Send()
				if err != nil {
					stats.Record(ctx, ClickHouseInsertErrors.M(1))
					cancel(fmt.Errorf("flushing batch: %w", err))
					return
				}
				batch, err = conn.PrepareBatch(ctx, insertStmt)
				if err != nil {
					cancel(fmt.Errorf("preparing new batch: %w", err))
					return
				}
			}
		case <-ctx.Done():
			if batch != nil && batch.Rows() > 0 {
				err = batch.Send()
				if err != nil {
					log.Error().Err(err).Msg("flushing final batch")
				}
				return
			}
		}
	}
}

func convertRow(report *Report) (*ClickhouseRow, error) {
	row := &ClickhouseRow{
		ID:        uint64(report.SequenceNumber),
		Time:      time.Unix(int64(report.FlowStartSeconds), 0),
		Frequency: uint32(report.Frequency),
		SNR:       int8(report.SNR),
		Mode:      report.Mode,
		Version:   report.ReceiverDecoderSoftware,
		RXSign:    report.ReceiverCallsign,
		RXLoc:     report.ReceiverLocator,
		TXSign:    report.SenderCallsign,
		TXLoc:     report.SenderLocator,
	}

	switch {
	case row.Frequency < 300e3:
		row.Band = -1
	case row.Frequency < 1e6:
		row.Band = 0
	default:
		row.Band = int16(row.Frequency / 1e6)
	}

	var rx_valid, tx_valid bool
	var rxloc, txloc maidenhead.Point

	if report.ReceiverLocator != "" {
		var err error
		rxloc, err = maidenhead.ParseLocatorCentered(report.ReceiverLocator)
		if err != nil {
			log.Warn().Err(err).Interface("report", report).Msg("parsing rxloc")
			return row, nil
		} else {
			row.RXLat, row.RXLon = float32(rxloc.Latitude), float32(rxloc.Longitude)
			rx_valid = true
		}
	}

	if report.SenderLocator != "" {
		var err error
		txloc, err = maidenhead.ParseLocatorCentered(report.SenderLocator)
		if err != nil {
			log.Warn().Err(err).Interface("report", report).Msg("parsing txloc")
			return row, nil
		} else {
			row.TXLat, row.TXLon = float32(txloc.Latitude), float32(txloc.Longitude)
			tx_valid = true
		}
	}

	if rx_valid && tx_valid {
		row.Distance = uint16(math.Round(txloc.Distance(rxloc)))
		row.Azimuth = uint16(math.Round(txloc.Bearing(rxloc)))
		if row.Azimuth == 360 {
			// LatLon.Azimuth normalizes to 0-360, but we might round 359.xx to 360
			row.Azimuth = 0
		}
		row.RXAzimuth = uint16(math.Round(rxloc.Bearing(txloc)))
		if row.RXAzimuth == 360 {
			row.RXAzimuth = 0
		}
	}
	return row, nil
}
