package main

import (
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const PREFIX = "github.com/arodland/pskhouse/"

var (
	PskReporterStatus = stats.Int64(PREFIX+"pskreporter/status",
		"PSKReporter stream status code",
		stats.UnitDimensionless,
	)

	StatusCode = tag.MustNewKey("http_status")

	PskReporterLinesRead = stats.Int64(PREFIX+"pskreporter/lines_read",
		"Lines read from PSKReporter",
		stats.UnitDimensionless,
	)

	PskReporterInvalidLines = stats.Int64(PREFIX+"pskreporter/invalid_lines",
		"Invalid lines from PSKReporter",
		stats.UnitDimensionless,
	)

	ClickHouseBatches = stats.Int64(PREFIX+"clickhouse/batches",
		"Batches sent to ClickHouse",
		stats.UnitDimensionless,
	)

	ClickHouseRows = stats.Int64(PREFIX+"clickhouse/rows",
		"Rows sent to ClickHouse",
		stats.UnitDimensionless,
	)

	ClickHouseBatchSize = stats.Float64(PREFIX+"clickhouse/batch_size",
		"Number of rows in a batch sent to ClickHouse",
		stats.UnitDimensionless,
	)

	ClickHouseConvertErrors = stats.Int64(PREFIX+"clickhouse/convert_errors",
		"Errors converting a report to ClickHouse",
		stats.UnitDimensionless,
	)

	ClickHouseInsertErrors = stats.Int64(PREFIX+"clickhouse/inert_errors",
		"Errors inserting a batch to ClickHouse",
		stats.UnitDimensionless,
	)
)

func initMetrics() {
	if err := view.Register(
		&view.View{
			Measure:     PskReporterStatus,
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{StatusCode},
		},
		&view.View{
			Measure:     PskReporterLinesRead,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure:     PskReporterInvalidLines,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure:     ClickHouseBatches,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure:     ClickHouseRows,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure:     ClickHouseConvertErrors,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure:     ClickHouseInsertErrors,
			Aggregation: view.Sum(),
		},
		&view.View{
			Measure: ClickHouseBatchSize,
			Aggregation: view.Distribution(
				1, 2, 3, 4, 5, 6, 7, 8, 9,
				10, 20, 30, 40, 50, 60, 70, 80, 90,
				100, 200, 300, 400, 500, 600, 700, 800, 900,
				1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000,
				10000,
			),
		},
	); err != nil {
		log.Fatal().Err(err).Msg("initializing metrics")
	}

	promReg := prom.NewRegistry()
	promReg.MustRegister(
		prom.NewProcessCollector(prom.ProcessCollectorOpts{}),
		prom.NewGoCollector(),
	)

	exporter, err := prometheus.NewExporter(prometheus.Options{Registry: promReg})
	if err != nil {
		log.Fatal().Err(err).Msg("initializing prom exporter")
	}
	view.RegisterExporter(exporter)

	http.Handle("/metrics", exporter)
}

func metricsServer() {
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.MetricsPort), nil)
	log.Fatal().Err(err).Send()
}
