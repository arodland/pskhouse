# PSKHouse

This is a daemon for importing the [PSK Reporter](https://pskreporter.info/) live stream into a ClickHouse database. The
concept is the same as the [wspr.live](https://wspr.live/) database, and the database schema is almost identical,
for easy porting of queries.

## Building

The loader is a [Go](https://go.dev/) program, and a binary can be built with `go build`. Alternatively, a docker image
can be built with [ko](https://ko.build/).

## ClickHouse Setup

This has been tested and run with the
[clickhouse/clickhouse-server](https://hub.docker.com/r/clickhouse/clickhouse-server):24.8-alpine image, but any vanilla
ClickHouse is probably fine to begin with. If using the docker image, make sure to set the `CLICKHOUSE_DB`,
`CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`, and `CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT` environment keys, or volume-mount a
configuration file with authentication enabled.

Before running the loader, create the table by running the SQL in `loader/schema.sql`.

## Configuration

All config flags can be provided either on the commandline (`-psk-reporter-token`) or in the environment
(`PSK_REPORTER_TOKEN`). It is recommended that secrets go in the environment so that they don't show up in `ps`.
Additionally, the clickhouse database, username, and password use the same environment keys as the
`clickhouse/clickhouse-server` docker image, so that the same environment-file can be used for both in that case.

* `-clickhouse-addr` / `CLICKHOUSE_ADDR`: host[:port] of the ClickHouse server to load data into.
* `-clickhouse-db` / `CLICKHOUSE_DB`: ClickHouse database name.
* `-clickhouse-username` / `CLICKHOUSE_USERNAME`: ClickHouse username.
* `-clickhouse-password` / `CLICKHOUSE_PASSWORD`: ClickHouse password.
* `-psk-reporter-token` / `PSK_REPORTER_TOKEN`: PSK Reporter stream token. This is an API key that must be requested
from Philip Gladstone.
* `-flush-frequency` / `FLUSH_FREQUENCY`: how often to flush a batch of reports to the database, in Go duration format (e.g. `1s`, `500ms`,
`1m`). The default of `1s` is probably fine. More frequent flushing increases ClickHouse CPU usage. Reports are also
flushed to the database on shutdown, unless the reason for the shutdown was a loss of database connection.
* `-metrics-port` / `METRICS_PORT`: port to run an HTTP server with a Prometheus-compatible `/metrics` endpoint.
* `-log-level` / `LOG_LEVEL`: minumum log level (verbosity): `trace`, `debug`, `info`, `warn`, `error`, `fatal`.

## Podman / Quadlet / Systemd

Unit files are included in `systemd` for a Linux system with [Podman](https://podman.io/) 5.0 or newer, which will run
the clickhouse server and the loader together in a pod and manage them as a systemd target `pskhouse.target`. In such a
case it expects the loader to exist as a local container image named `pskhouse-loader`, and it can be built using ko. If
ClickHouse is being managed another way, it's perfectly fine to forego all of this and run the Go binary directly.

## Examples!

![A query showing most recent reports](https://github.com/user-attachments/assets/0c7e30b2-464e-4ccf-a4f7-cc1cf45aa799)


![A query showing top spotters of the past 24 hours](https://github.com/user-attachments/assets/e7ecb663-37fb-4f42-aedf-ac9c44d8aff8)
