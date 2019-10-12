package adapter

import (
	"context"
	"sync"
	"time"

	"github.com/gotwarlost/chaos-adapter/adapter/chaos"
	"github.com/gotwarlost/chaos-adapter/util"
	model "istio.io/api/mixer/adapter/model/v1beta1"
	rpc "istio.io/gogo-genproto/googleapis/google/rpc"
	"istio.io/istio/mixer/pkg/status"
)

// ChaosAdapter is the base class for our adapter
type ChaosAdapter struct {
	l     sync.Mutex
	delay time.Duration
}

func (c *ChaosAdapter) SetDelay(d time.Duration) {
	c.l.Lock()
	defer c.l.Unlock()
	c.delay = d
}

func (c *ChaosAdapter) Delay() time.Duration {
	c.l.Lock()
	defer c.l.Unlock()
	return c.delay
}

func (c *ChaosAdapter) nextDelay() time.Duration {
	d := c.Delay()
	if d == 0 {
		return 0
	}
	return util.NextDelay(d)
}

// HandleChaos handles each request
func (c *ChaosAdapter) HandleChaos(ctx context.Context, req *chaos.HandleChaosRequest) (*model.CheckResult, error) {
	time.Sleep(c.nextDelay())
	if req.Instance.Hello == "" {
		return &model.CheckResult{
			ValidDuration: time.Nanosecond,
			Status:        status.New(rpc.UNAUTHENTICATED),
		}, nil
	}
	return &model.CheckResult{
		ValidDuration: time.Nanosecond,
		Status:        status.New(rpc.OK),
	}, nil
}
