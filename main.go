package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	appVersion string
	version    = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "demo_version",
		Help: "Version information about this binary",
		ConstLabels: map[string]string{
			"version": appVersion,
		},
	})

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "demo_http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "demo_http_request_duration_seconds",
		Help: "Duration of all HTTP requests",
	}, []string{"code", "handler", "method"})
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var s0 string

func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func main() {
	version.Set(1)
	bind := ""
	enableH2c := false
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&bind, "bind", ":8080", "The socket to bind to.")
	flagset.BoolVar(&enableH2c, "h2c", false, "Enable h2c (http/2 over tcp) protocol.")
	flagset.Parse(os.Args[1:])

	r := prometheus.NewRegistry()
	r.MustRegister(httpRequestsTotal)
	r.MustRegister(httpRequestDuration)
	r.MustRegister(version)

	foundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		w.Write([]byte("Hello from example application."))
	})
	notfoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		w.WriteHeader(http.StatusNotFound)
	})

	memLeakHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		w.WriteHeader(http.StatusOK)
		s0 = RandStringBytes(10000000)

	})

	foundChain := promhttp.InstrumentHandlerDuration(
		httpRequestDuration.MustCurryWith(prometheus.Labels{"handler": "found"}),
		promhttp.InstrumentHandlerCounter(httpRequestsTotal, foundHandler),
	)

	mux := http.NewServeMux()
	mux.Handle("/", foundChain)
	mux.Handle("/err", promhttp.InstrumentHandlerCounter(httpRequestsTotal, notfoundHandler))
	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	mux.Handle("/mem", memLeakHandler)

	var srv *http.Server
	if enableH2c {
		srv = &http.Server{Addr: bind, Handler: h2c.NewHandler(mux, &http2.Server{})}
	} else {
		srv = &http.Server{Addr: bind, Handler: mux}
	}

	log.Fatal(srv.ListenAndServe())
}
