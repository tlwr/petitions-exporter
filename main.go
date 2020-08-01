package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	nlogrus "github.com/meatballhat/negroni-logrus"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sethvargo/go-signalcontext"
	"github.com/sirupsen/logrus"
	nsecure "github.com/unrolled/secure"
	"github.com/urfave/negroni"
	nprom "github.com/zbindenren/negroni-prometheus"

	"github.com/tlwr/petitions-exporter/pkg/petitions-fetcher"
)

func main() {
	petitionsURL := os.Getenv("PETITIONS_URL")
	if petitionsURL == "" {
		petitionsURL = "https://petition.parliament.uk"
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	f := fetcher.New(petitionsURL, 10*time.Minute, logger)
	f.Start()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "healthy")
	})

	mux.Handle("/metrics", promhttp.Handler())

	n := negroni.New()
	n.Use(negroni.NewRecovery())
	n.Use(nlogrus.NewMiddlewareFromLogger(logger, "web"))
	n.Use(gzip.Gzip(gzip.DefaultCompression))
	n.Use(negroni.HandlerFunc(nsecure.New().HandlerFuncWithNext))
	n.Use(nprom.NewMiddleware("petitions-exporter"))
	n.UseHandler(mux)

	ctx, cancel := signalcontext.On(syscall.SIGTERM)
	defer cancel()

	server := &http.Server{Addr: ":8080", Handler: n}

	go func() {
		server.ListenAndServe()
	}()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	f.Stop()
	f.Wait()

	os.Exit(0)
}
