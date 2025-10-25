package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/PapyrusVIP/isomer/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func metrics(e *env, args ...string) error {
	set := e.newFlagSet("metrics", "address", "port", "--", "path")
	set.Description = `
		Expose metrics in prometheus export format.

		Examples:
		  $ isomctl metrics 127.0.0.1 8000
		  THEN
		  $ curl http://127.0.0.1:8000/metrics`

	timeout := set.Duration("timeout", 30*time.Second, "Duration to wait for an HTTP metrics request to complete.")
	if err := set.Parse(args); err != nil {
		return err
	}

	address := set.Arg(0)
	port := set.Arg(1)
	path := set.Arg(2)

	if path == "" {
		path = "/"
	}

	if err := e.setupEnv(); err != nil {
		return err
	}

	// Create an instance of the prometheus registry and register all collectors.
	reg, err := isomerRegistry(e)
	if err != nil {
		return err
	}

	// Create TCP listener used for metrics endpoint.
	ln, err := e.listen("tcp", fmt.Sprintf("%s:%s", address, port))
	if err != nil {
		return err
	}
	defer ln.Close()

	e.stdout.Log("Listening on", ln.Addr().String())

	// Create an instance of the metrics server
	srv := metricsServer(e.ctx, reg, path, timeout)

	// Close the http server when the env context is closed.
	go func() {
		<-e.ctx.Done()
		srv.Close()
	}()

	// Block on serving the metrics http server.
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %s", err)
	}

	return nil
}

func isomerRegistry(e *env) (*prometheus.Registry, error) {
	reg := prometheus.NewRegistry()
	isomerReg := prometheus.WrapRegistererWithPrefix("isomer_", reg)

	coll := internal.NewCollector(e.stderr, e.netns, e.bpfFs)
	if err := isomerReg.Register(coll); err != nil {
		return nil, fmt.Errorf("register collector: %s", err)
	}

	buildInfo := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "build_info",
		Help: "Build and version information",
		ConstLabels: prometheus.Labels{
			"goversion": runtime.Version(),
			"version":   Version,
		},
	})
	buildInfo.Set(1)
	if err := reg.Register(buildInfo); err != nil {
		return nil, fmt.Errorf("register build info: %s", err)
	}
	return reg, nil
}

func metricsServer(ctx context.Context, reg *prometheus.Registry, path string, t *time.Duration) http.Server {
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorHandling:       promhttp.HTTPErrorOnError,
		MaxRequestsInFlight: 1,
		Timeout:             *t,
		
	})

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	return http.Server{
		Handler:     mux,
		ReadTimeout: *t,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
}

