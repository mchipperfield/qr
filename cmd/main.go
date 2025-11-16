package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"github.com/mchipperfield/qr"
	"github.com/mchipperfield/qr/api"
	"golang.org/x/exp/slog"
)

var (
	service string = "qrcode"
	version string = "unknown_version"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	version = info.Main.Version
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	host, err := os.Hostname()
	if err != nil {
		logger.Error("unable to resolve host", "error", err)
		os.Exit(1)
	}

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	var (
		port            = flags.Int("port", 8080, "port for api server to listen on")
		env             = flags.String("env", "dev", "deployment environment")
		shutdownTimeout = flags.Duration("shutdown_timeout", 10*time.Second, "time to allow for graceful shutdown")
	)
	if err := flags.Parse(os.Args[1:]); err != nil {
		logger.Info("failed to parse flags", "error", err)
		os.Exit(1)
	}

	logger = logger.With(slog.Group("service", "name", service, "host", host, "version", version, "environment", *env))
	logger.Info("starting service")
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Warn("recovered from panic", "error", err)
					return
				}
			}()

			metrics := httpsnoop.CaptureMetrics(next, w, r)
			logger.Info("http request", slog.Group("request", "method", r.Method, "uri", r.RequestURI, "proto", r.Proto, "code", metrics.Code, "size", metrics.Written))
		})
	}

	router := mux.NewRouter().StrictSlash(true)
	router.Path("/").Handler(qr.NewHandler(logger))
	router.PathPrefix("/v1/").Handler(http.StripPrefix("/v1", api.NewHandler(logger)))
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mw(router),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	errorChan := make(chan error)

	go func() {
		logger.Info("starting http server", "port", srv.Addr)
		errorChan <- srv.ListenAndServe()
	}()

	select {
	case err := <-errorChan:
		logger.Info("listen and serve error", "error", err)
	case stopSig := <-stopChan:
		logger.Info("shutdown signal received", "signal", stopSig.String())
		ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
		defer cancel()
		logger.Info("shutting down http server", "port", srv.Addr)
		if err := srv.Shutdown(ctx); err != nil {
			logger.Info("failed to gracefully shutdown", "error", err)
			srv.Close()
		}
	}
}
