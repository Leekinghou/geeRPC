package main

import (
	"fmt"
	client2 "geeRPC/client"
	geerpc "geeRPC/codec"
	"log"
	"net"
	"sync"
	"time"
)

func startServer(addr chan string) {
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
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	// 等待所有请求处理完成
	wg.Wait()
}
