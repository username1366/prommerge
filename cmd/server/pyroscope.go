package main

import (
	"github.com/grafana/pyroscope-go"
	"log/slog"
)

func Pyroscope() {
	slog.Info("Start pyroscope profiler")
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "prommerge",

		// replace this with the address of pyroscope server
		ServerAddress: "http://127.1:4040",

		// you can disable logging by setting this to nil
		Logger: pyroscope.StandardLogger,

		// Optional HTTP Basic authentication (Grafana Cloud)
		// Optional Pyroscope tenant ID (only needed if using multi-tenancy). Not needed for Grafana Cloud.
		// TenantID:          "<TenantID>",

		// by default all profilers are enabled,
		// but you can select the ones you want to use:
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
}
