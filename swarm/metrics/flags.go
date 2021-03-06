// Authored and revised by YOC team, 2018
// License placeholder #1

package metrics

import (
	"time"

	"github.com/Yocoin15/Yocoin_Sources/cmd/utils"
	yocoinmetrics "github.com/Yocoin15/Yocoin_Sources/metrics"
	"github.com/Yocoin15/Yocoin_Sources/metrics/influxdb"
	"github.com/Yocoin15/Yocoin_Sources/swarm/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	metricsEnableInfluxDBExportFlag = cli.BoolFlag{
		Name:  "metrics.influxdb.export",
		Usage: "Enable metrics export/push to an external InfluxDB database",
	}
	metricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "Metrics InfluxDB endpoint",
		Value: "http://127.0.0.1:8086",
	}
	metricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "Metrics InfluxDB database",
		Value: "metrics",
	}
	metricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "Metrics InfluxDB username",
		Value: "",
	}
	metricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "Metrics InfluxDB password",
		Value: "",
	}
	// The `host` tag is part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// It is used so that we can group all nodes and average a measurement across all of them, but also so
	// that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	metricsInfluxDBHostTagFlag = cli.StringFlag{
		Name:  "metrics.influxdb.host.tag",
		Usage: "Metrics InfluxDB `host` tag attached to all measurements",
		Value: "localhost",
	}
)

// Flags holds all command-line flags required for metrics collection.
var Flags = []cli.Flag{
	utils.MetricsEnabledFlag,
	metricsEnableInfluxDBExportFlag,
	metricsInfluxDBEndpointFlag, metricsInfluxDBDatabaseFlag, metricsInfluxDBUsernameFlag, metricsInfluxDBPasswordFlag, metricsInfluxDBHostTagFlag,
}

func Setup(ctx *cli.Context) {
	if yocoinmetrics.Enabled {
		log.Info("Enabling swarm metrics collection")
		var (
			enableExport = ctx.GlobalBool(metricsEnableInfluxDBExportFlag.Name)
			endpoint     = ctx.GlobalString(metricsInfluxDBEndpointFlag.Name)
			database     = ctx.GlobalString(metricsInfluxDBDatabaseFlag.Name)
			username     = ctx.GlobalString(metricsInfluxDBUsernameFlag.Name)
			password     = ctx.GlobalString(metricsInfluxDBPasswordFlag.Name)
			hosttag      = ctx.GlobalString(metricsInfluxDBHostTagFlag.Name)
		)

		// Start system runtime metrics collection
		go yocoinmetrics.CollectProcessMetrics(2 * time.Second)

		if enableExport {
			log.Info("Enabling swarm metrics export to InfluxDB")
			go influxdb.InfluxDBWithTags(yocoinmetrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "swarm.", map[string]string{
				"host": hosttag,
			})
		}
	}
}
