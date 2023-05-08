package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"geeRPC/codec/codec"
	"geeRPC/service"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

/**
GeeRPC 客户端最核心的部分 Client
*/

// Client closing 和 shutdown 任意一个值置为 true，则表示 Client 处于不可用的状态，
// 但有些许的差别，closing 是用户主动关闭的，即调用 Close 方法，而 shutdown 置为 true 一般是有错误发生。
type Client struct {
	cc       codec.Codec      // cc 是消息的编解码器，用来序列化将要发送出去的请求，以及反序列化接收到的响应。
	opt      *service.Option  // opt 是 Option 的一个指针，包含了 Option 中的所有选项。
	sending  sync.Mutex       // sending 是一个互斥锁，为了保证请求的有序发送，即防止出现多个请求报文混淆。
	header   codec.Header     // header 是每一个请求独有的，因此每次请求都要拥有独立的 header。
	mu       sync.Mutex       // mu 用来保护 pending 字典。
	seq      uint64           // 每个请求拥有唯一编号
	pending  map[uint64]*Call // 存储未处理完的请求，键是编号，值是 Call 实例
	closing  bool             // user has called Close
	shutdown bool             // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shutdown")

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable 判断连接是否可用
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

//// Dial Dial 函数，便于用户传入服务端地址，创建 Client 实例。
//func Dial(network, address string, opts ...*service.Option) (client *Client, err error) {
//	opt, err := parseOptions(opts...)
//	if err != nil {
//		return nil, err
//	}
//	conn, err := net.Dial(network, address)
//	if err != nil {
//		return nil, err
//	}
//	// close the connection if client is nil
//	defer func() {
//		if client == nil {
//			_ = conn.Close()
//		}
//	}()
//	return NewClient(conn, opt)
//}

func NewClient(conn net.Conn, opt *service.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *service.Option) *Client {
	client := &Client{
		seq:     1, // seq starts with 1, 0 means invalid call
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	// 创建一个子协程调用 receive() 接收响应
	go client.receive()
	return client
}

func parseOptions(opts ...*service.Option) (*service.Option, error) {
	// if opts is nil or pass nil as parameter
	if len(opts) == 0 || opts[0] == nil {
		return service.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = service.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = service.DefaultOption.CodecType
	}
	return opt, nil
}

// Go Go异步调用函数。
// 返回表示调用的Call结构。
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// Call 调用命名函数，等待它完成;
// 返回错误状态
//func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
//	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
//	return call.Error
//}

// Go 和 Call 是客户端暴露给用户的两个 RPC 服务调用接口，Go 是一个异步接口，返回 call 实例。
// Call 是对 Go 的封装，阻塞 call.Done，等待响应返回，是一个同步接口。

//type newClientFunc func(conn net.Conn, opt *service.Option) (client *Client, err error)

func NewHTTPClient(conn net.Conn, opt *service.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", "/_geerpc_"))

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == "200 Connected to Gee RPC" {
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

// DialHTTP connects to an HTTP RPC server at the specified network address
// listening on the default HTTP RPC path.
func DialHTTP(network, address string, opts ...*service.Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

// XDial calls different functions to connect to a RPC server
// according the first parameter rpcAddr.
// rpcAddr is a general format (protocol@addr) to represent a rpc server
// eg, http@10.0.0.1:7001, tcp@10.0.0.1:9999, unix@/tmp/geerpc.sock
func XDial(rpcAddr string, opts ...*service.Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		// tcp, unix or other transport protocol
		return Dial(protocol, addr, opts...)
	}
}
