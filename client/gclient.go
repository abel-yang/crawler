package main

import (
	"context"
	"fmt"
	pb "github.com/abel-yang/crawler/proto/greeter"
	grpccli "github.com/go-micro/plugins/v4/client/grpc"
	"github.com/go-micro/plugins/v4/registry/etcd"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
)

func main() {
	reg := etcd.NewRegistry(
		registry.Addrs(":2379"))

	service := micro.NewService(
		micro.Registry(reg),
		micro.Client(grpccli.NewClient()),
	)

	//parse command line flags
	service.Init()

	//use the generated client sub
	cl := pb.NewGreeterService("go.micro.server.worker", service.Client())

	//make request
	rsp, err := cl.Hello(context.Background(), &pb.Request{
		Name: "abel",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp.Greeting)
}
