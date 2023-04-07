package main

import (
	"context"
	"demo-services/services/service1/api/hello"
	"demo-services/utils"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
)

const (
	consulAddress = "127.0.0.1:8500"
	localPort     = 3001
)

func client(name, tag string) hello.HelloServiceClient {
	var lastIndex uint64
	config := consulapi.DefaultConfig()
	config.Address = consulAddress //consul server

	client, err := consulapi.NewClient(config)
	if err != nil {
		fmt.Println("api new client is failed, err:", err)
		return nil
	}
	services, metainfo, err := client.Health().Service(name, tag, true, &consulapi.QueryOptions{
		WaitIndex: lastIndex, // 同步点，这个调用将一直阻塞，直到有新的更新
	})
	if err != nil {
		fmt.Println(err)
	}
	lastIndex = metainfo.LastIndex

	for _, service := range services {
		fmt.Println("service.Service.Address:", service.Service.Address, "service.Service.Port:", service.Service.Port)
	}
	if len(services) > 0 {
		addr := fmt.Sprintf("%s:%d", services[0].Service.Address, services[0].Service.Port)

		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		c := hello.NewHelloServiceClient(conn)
		return c
	}

	return nil
}

func consulRegister() {
	// 创建连接consul服务配置
	config := consulapi.DefaultConfig()
	config.Address = consulAddress
	client, err := consulapi.NewClient(config)
	if err != nil {
		fmt.Println("consul client error : ", err)
	}

	// 创建注册到consul的服务到
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = "sv2"
	registration.Name = "sv2" //根据这个名称来找这个服务
	registration.Port = localPort
	registration.Tags = []string{"sv2"} //这个就是一个标签，可以根据这个来找这个服务，相当于V1.1这种
	registration.Address = utils.LocalIP()

	// 增加consul健康检查回调函数
	check := new(consulapi.AgentServiceCheck)
	check.HTTP = fmt.Sprintf("http://%s:%d", registration.Address, registration.Port)
	check.Timeout = "5s"                         //超时
	check.Interval = "5s"                        //健康检查频率
	check.DeregisterCriticalServiceAfter = "30s" // 故障检查失败30s后 consul自动将注册服务删除
	registration.Check = check

	// 注册服务到consul
	err = client.Agent().ServiceRegister(registration)
}

// Handler 3001
func Handler(w http.ResponseWriter, r *http.Request) {
	c := client("sv1", "sv1")
	if c == nil {
		w.Write([]byte("has not get a service named sv1"))
	} else {
		rep, _ := c.SayHello(context.Background(), &hello.Req{
			Name: "hello",
		})
		w.Write([]byte(rep.Content))
	}

}

// ServerLoad 启动
func ServerLoad() {
	consulRegister()
	//定义一个http接口
	http.HandleFunc("/", Handler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", localPort), nil)
	if err != nil {
		fmt.Println("error: ", err.Error())
	}
}

func main() {
	ServerLoad()
}
