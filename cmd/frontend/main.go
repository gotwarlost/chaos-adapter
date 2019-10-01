package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dimfeld/httptreemux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const ns = "chaos_frontend_client"

var counter *prometheus.CounterVec
var histo *prometheus.HistogramVec

func init() {
	counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "request_count",
		Help:      "total chaos client requests",
	}, []string{"code"})
	prometheus.MustRegister(counter)
	histo = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "latency",
		Help:      "chaos client request latency",
	}, []string{"code"})
	prometheus.MustRegister(histo)
}

type stats struct {
	l                    sync.Mutex  `json:"-"`
	ConnectionErrorCount int         `json:"connectionErrorCount"`
	SuccessCount         int         `json:"successCount"`
	ResponseCodes        map[int]int `json:"responseCodes"`
}

func secs(d time.Duration) float64 {
	return float64(d) / float64(time.Second)
}

func (s *stats) incConnectionError(err error, d time.Duration) {
	log.Println("connection error", err)
	s.l.Lock()
	defer s.l.Unlock()
	s.ConnectionErrorCount++
	counter.WithLabelValues("CONN").Inc()
	histo.WithLabelValues("CONN").Observe(secs(d))
}

func (s *stats) incResult(resCode int, d time.Duration) {
	s.l.Lock()
	defer s.l.Unlock()
	if resCode < 500 {
		s.SuccessCount++
	}
	s.ResponseCodes[resCode]++
	l := fmt.Sprintf("%d", resCode)
	counter.WithLabelValues(l).Inc()
	histo.WithLabelValues(l).Observe(secs(d))
}

func (s *stats) JSON() []byte {
	s.l.Lock()
	defer s.l.Unlock()
	b, _ := json.MarshalIndent(s, "", "  ")
	return b
}

var clientStats = &stats{ResponseCodes: map[int]int{}}

func httpHandler() http.Handler {
	mux := httptreemux.New()
	mux.GET("/status", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		w.Write(clientStats.JSON())
	})
	mh := promhttp.Handler()
	mux.GET("/metrics", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		mh.ServeHTTP(w, r)
	})
	return mux
}

func main() {
	var (
		serverURL   = "http://chaos-server"
		listenPort  = 8080
		concurrency = 10
		header      = "X-Hello"
		timeoutSecs = 1
		delayMillis = 100
	)

	flag.StringVar(&serverURL, "server", serverURL, "server URL")
	flag.IntVar(&listenPort, "listen-port", listenPort, "listen port")
	flag.IntVar(&concurrency, "concurrency", concurrency, "request concurrency")
	flag.IntVar(&timeoutSecs, "timeout", timeoutSecs, "request timeout in seconds")
	flag.IntVar(&delayMillis, "delay", delayMillis, "delay between requests in milliseconds")
	flag.StringVar(&header, "header-name", header, "header to set")

	done := make(chan struct{})
	doHTTP := func(client *http.Client) {
		start := time.Now()
		req, err := http.NewRequest(http.MethodGet, serverURL, nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set(header, "world")
		res, err := client.Do(req)
		if err != nil {
			clientStats.incConnectionError(err, time.Now().Sub(start))
			return
		}
		defer res.Body.Close()
		_, _ = ioutil.ReadAll(res.Body)
		clientStats.incResult(res.StatusCode, time.Now().Sub(start))
	}
	worker := func() {
		client := &http.Client{Timeout: time.Duration(timeoutSecs) * time.Second}
		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(time.Duration(delayMillis) * time.Millisecond)
				doHTTP(client)
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker()
		}()
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: httpHandler(),
	}
	err := server.ListenAndServe()
	close(done)
	wg.Wait()
	if err != nil {
		log.Fatalln(err)
	}
}
