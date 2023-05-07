package service

import (
	"encoding/json"
	"geeRPC/codec/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
type Server struct {
	serviceMap sync.Map
}

// request stores all information of a call
type request struct {
	header       *codec.Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer Accept accepts connections on the listener and serves requests
var DefaultServer = NewServer()

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	// for 循环等待 socket 连接建立
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error: ", err)
			return
		}
		// 开启子协程处理，处理过程交给了 ServerConn 方法
		go server.ServeConn(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// ServeConn ServeConn在单连接上运行服务器。
// ServeConn 阻塞，服务连接，直到客户端挂起。
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	// json.NewDecoder 反序列化得到 Option 实例
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	// 根据 CodeType 得到对应的消息编解码器，接下来的处理交给 serverCodec
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

//serveCodec 的过程非常简单。主要包含三个阶段
//读取请求 readRequest
//处理请求 handleRequest
//回复请求 sendResponse
func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // 确保发送完整的响应
	wg := new(sync.WaitGroup)  // wait until all request are handled
	// 在一次连接中，允许接收多个请求，即多个 request header 和 request body，因此使用for循环无限等待请求到来
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // 退出循环
			}
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// handleRequest 使用了协程并发执行请求
		// 处理请求是并发的，但是回复请求的报文必须是逐个发送的，使用锁(sending)保证。
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var header codec.Header
	if err := cc.ReadHeader(&header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &header, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{header: header}
	req.svc, req.mtype, err = server.findService(header.ServiceMethod)
	if err != nil {
		return req, err
	}
	// 1. 最重要的内容：newArgv() 和 newReplyv() 两个方法创建出两个入参实例
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// make sure that argvi is a pointer, ReadBody need a pointer as parameter
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	// 2. 通过 cc.ReadBody() 将请求报文反序列化为第一个入参 argv
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	// 通过 req.svc.call 完成方法调用，将 replyv 传递给 sendResponse 完成序列化即可。
	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.header.Error = err.Error()
		server.sendResponse(cc, req.header, invalidRequest, sending)
		return
	}
	server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
}
