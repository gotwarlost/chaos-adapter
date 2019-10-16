package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dimfeld/httptreemux"
	"github.com/gotwarlost/chaos-adapter/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const ns = "chaos_backend"

var counter prometheus.Counter
var histo prometheus.Histogram

func init() {
	counter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "request_count",
		Help:      "total chaos server requests",
	})
	prometheus.MustRegister(counter)
	histo = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "latency",
		Help:      "chaos server request latency",
	})
	prometheus.MustRegister(histo)
}

var l sync.Mutex
var d time.Duration

func nextDelay() time.Duration {
	l.Lock()
	defer l.Unlock()
	if d == 0 {
		return 0
	}
	return util.NextDelay(d)
}

func setDelay(newAvg time.Duration) {
	l.Lock()
	defer l.Unlock()
	d = newAvg
}

func httpHandler() http.Handler {
	mux := httptreemux.New()
	mux.Handle(http.MethodGet, "/status", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		_, _ = io.WriteString(w, "OK\n")
	})
	mux.Handle(http.MethodPost, "/delay/:delay", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		x := params["delay"]
		delay, err := time.ParseDuration(x)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid delay: %s, %v", x, err), http.StatusBadRequest)
		}
		setDelay(delay)
		w.WriteHeader(http.StatusNoContent)
	})
	metricsHandler := promhttp.Handler()
	mux.Handle(http.MethodGet, "/metrics", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		metricsHandler.ServeHTTP(w, r)
	})
	mux.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(nextDelay())
		_, _ = io.WriteString(w, fmt.Sprintf("hello %s\n", r.Header.Get("X-Hello")))
		d := float64(time.Now().Sub(start)) / float64(time.Second)
		counter.Inc()
		histo.Observe(d)
	}
	return mux
}

func main() {
	var port int
	var defaultDelay string
	var exitDelay string
	flag.StringVar(&defaultDelay, "delay", "0", "default mean delay for response")
	flag.StringVar(&exitDelay, "exit-delay", "0", "default delay for server exit")
	flag.IntVar(&port, "port", 8080, "listen port")
	flag.Parse()
	d, err := time.ParseDuration(defaultDelay)
	if err != nil {
		log.Fatalln(err)
	}
	setDelay(d)

	ed, err := time.ParseDuration(exitDelay)
	if err != nil {
		log.Fatalln(err)
	}

	stop := func() {
		log.Println("Got SIGTERM, stopping in", ed)
		time.Sleep(ed)
		os.Exit(1)
	}
	var once sync.Once
	go func() {
		ch := make(chan os.Signal, 5)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		for range ch {
			once.Do(stop)
		}
	}()

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), httpHandler()); err != nil {
		log.Fatalln(err)
	}
}
