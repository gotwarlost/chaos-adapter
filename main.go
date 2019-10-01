package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dimfeld/httptreemux"
	"github.com/gotwarlost/chaos-adapter/adapter"
	"github.com/gotwarlost/chaos-adapter/adapter/chaos"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var instance = &adapter.ChaosAdapter{}

type grpcRunner struct {
	port int
	l    sync.Mutex
	s    *grpc.Server
}

func (g *grpcRunner) isRunning() bool {
	g.l.Lock()
	defer g.l.Unlock()
	return g.s != nil
}

func (g *grpcRunner) end() error {
	g.l.Lock()
	defer g.l.Unlock()
	if g.s == nil {
		return fmt.Errorf("server not running")
	}
	log.Println("stopping grpc server")
	g.s.GracefulStop()
	g.s = nil
	return nil
}

func (g *grpcRunner) start() error {
	g.l.Lock()
	defer g.l.Unlock()
	if g.s != nil {
		return fmt.Errorf("server already running")
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(adapter.Middleware()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: 1 * time.Minute,
		}))
	chaos.RegisterHandleChaosServiceServer(s, instance)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", g.port))
	if err != nil {
		return err
	}

	g.s = s
	go func() {
		log.Println("starting grpc server", "address", l.Addr())
		if err := s.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Println("grpc server failed", err)
			g.end()
		}
	}()
	return nil
}

var grpcInstance *grpcRunner

func httpHandler() http.Handler {
	mux := httptreemux.New()
	mux.POST("/grpc/delay/:duration", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		dstr := p["duration"]
		d, err := time.ParseDuration(dstr)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid duration %q, %v", dstr, err), http.StatusBadRequest)
			return
		}
		instance.SetDelay(d)
		_, _ = fmt.Fprintf(w, "grpc server delay set to %s\n", dstr)
	})
	mux.POST("/grpc/start", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		err := grpcInstance.start()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("grpc server started\n"))
	})
	mux.POST("/grpc/stop", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		err := grpcInstance.end()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("grpc server stopped\n"))
	})
	mux.GET("/status", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		running := grpcInstance.isRunning()
		delay := instance.Delay().Truncate(time.Millisecond).String()
		_, _ = fmt.Fprintf(w, "grpc running: %v\ndelay: %s\n", running, delay)
	})
	metricsHandler := promhttp.Handler()
	mux.GET("/metrics", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		metricsHandler.ServeHTTP(w, r)
	})
	return mux
}

func main() {
	grpcPort := 4080
	httpPort := 8080
	grpcInstance = &grpcRunner{port: grpcPort}
	if err := grpcInstance.start(); err != nil {
		log.Fatalln(err)
	}
	defer grpcInstance.end()

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: httpHandler(),
	}
	log.Println("starting HTTP server", "address", s.Addr)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
