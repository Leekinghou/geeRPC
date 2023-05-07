package main

import (
	client2 "geeRPC/client"
	geerpc "geeRPC/service"
	"log"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	// 注册 Foo 到 Server 中，并启动 RPC 服务
	var foo Foo
	if err := geerpc.Register(&foo); err != nil {
		log.Fatal("register error: ", err)
	}
	listen, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start rpc server on", listen.Addr())
	// 使用了信道 addr，确保服务端端口监听成功，客户端再发起请求。
	addr <- listen.Addr().String()
	geerpc.Accept(listen)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	client, _ := client2.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	//  并发 5 个 RPC 同步调用
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	// 等待所有请求处理完成
	wg.Wait()
}
