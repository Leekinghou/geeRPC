package main

import (
	"encoding/json"
	"fmt"
	geerpc "geeRPC/codec"
	"geeRPC/codec/codec"
	"log"
	"net"
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
	addr := make(chan string)
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr) // 阻塞，等待服务器启动完毕
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// 客户端首先发送 Option 进行协议交换，接下来发送消息头 h := &codec.Header{}，和消息体 geerpc req ${h.Seq}。
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("geerpc req: %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}
