package main

import (
	"context"
	"demo-services/services/service1/api/hello"
	"demo-services/utils"
	"fmt"
	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"log"
	"net"
)

type HealthImpl struct{}

// Check 实现健康检查接口，这里直接返回健康状态，这里也可以有更复杂的健康检查策略，比如根据服务器负载来返回
func (h *HealthImpl) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch 这个没用，只是为了让HealthImpl实现RegisterHealthServer内部的interface接口, 监听服务变化
func (h *HealthImpl) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return nil
}

type HelloServiceServer struct {
	hello.HelloServiceServer
}

func (h *HelloServiceServer) SayHello(ctx context.Context, req *hello.Req) (*hello.Response, error) {
	return &hello.Response{
		Content: req.Name + " world",
	}, nil
}

const (
	consulAddress = "127.0.0.1:8500"
	localPort     = 3002
)

func externalServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	hello.RegisterHelloServiceServer(s, &HelloServiceServer{})
	grpc_health_v1.RegisterHealthServer(s, &HealthImpl{}) //比普通的grpc开启多了这一步
	s.Serve(lis)
	log.Println("grpc start")
}

func grpcRegister() {
	config := api.DefaultConfig()
	config.Address = consulAddress
	client, err := api.NewClient(config)
	if err != nil {
		panic(err)
	}
	agent := client.Agent()
	localIP := utils.LocalIP()
	reg := &api.AgentServiceRegistration{
		ID:      "sv1",           // 服务节点的名称
		Name:    "sv1",           // 服务名称
		Tags:    []string{"sv1"}, // tag，可以为空
		Port:    localPort,       // 服务端口
		Address: localIP,         // 服务 IP
		Check: &api.AgentServiceCheck{ // 健康检查
			Interval: "5s", // 健康检查间隔
			// grpc 支持，执行健康检查的地址，service 会传到 Health.Check 函数中
			GRPC:                           fmt.Sprintf("%v:%v/%v", localIP, localPort, "hello"),
			DeregisterCriticalServiceAfter: "30s", // 注销时间，相当于过期时间
		},
	}
	if err := agent.ServiceRegister(reg); err != nil {
		panic(err)
	}
}

func main() {
	grpcRegister()
	externalServer()
}
