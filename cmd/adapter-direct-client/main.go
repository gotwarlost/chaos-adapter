package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/gotwarlost/chaos-adapter/adapter/chaos"
	"google.golang.org/grpc"
)

func main() {
	cc, err := grpc.Dial("localhost:4080", grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

outer:
	for {
		select {
		case <-sigChan:
			break outer
		default:
			break
		}
		client := chaos.NewHandleChaosServiceClient(cc)
		result, err := client.HandleChaos(context.TODO(), &chaos.HandleChaosRequest{
			Instance: &chaos.InstanceMsg{
				Name:  "chaos-adapter",
				Hello: "World",
			},
		},
			grpc.WaitForReady(true),
		)
		if err != nil {
			log.Println(reflect.TypeOf(err))
			log.Printf("%v\n", reflect.TypeOf(err))
			log.Fatalln(err)
		}
		log.Println(result)
		time.Sleep(time.Second)
	}

}
