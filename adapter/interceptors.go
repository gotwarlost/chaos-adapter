package adapter

import (
	"context"
	"fmt"
	"log"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	model "istio.io/api/mixer/adapter/model/v1beta1"
)

const ns = "chaos_adapter"

var counter prometheus.Counter
var histo prometheus.Histogram

func init() {
	counter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: ns,
		Name:      "request_count",
		Help:      "total chaos adapter requests",
	})
	prometheus.MustRegister(counter)
	histo = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "latency",
		Help:      "chaos adapter request latency",
	})
	prometheus.MustRegister(histo)
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	durationStringMS := fmt.Sprintf("%0.3f", duration.Seconds()*1000)
	checkResp := resp.(*model.CheckResult)

	counter.Inc()
	histo.Observe(float64(duration) / float64(time.Second))

	log.Println("request: code=", codes.Code(checkResp.Status.Code), "status=", checkResp.Status.Message, "duration-ms=", durationStringMS)
	return resp, err
}

// return interceptor chain
func Middleware() grpc.UnaryServerInterceptor {
	return grpc_middleware.ChainUnaryServer(
		loggingInterceptor,
	)
}
