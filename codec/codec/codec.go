package codec

import "io"

// Header 一个典型的 RPC 调用如下：
// err = client.Call("Arith.Multiply", args, &reply)
// 客户端发送的请求包括: 1. 服务名 Arith 2. 方法名 Multiply 3. 参数 args 三个
// 服务端的响应包括: 1. 错误error 2. 返回值 reply
// 我们将请求和响应中的参数和返回值抽象为body， 剩余的信息放在header中，那么就可以抽象出数据结构 Header：
type Header struct {
	ServiceMethod string // 服务名和方法名，与Go语言中的结构体和方法相映射。format "Service.Method"
	Seq           uint64 // 请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求 sequence number chosen by client
	Error         string // 客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中
}

// Codec 抽象出对消息体进行编解码的接口 Codec
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFunc 抽象出 Codec 的构造函数
// 定义一个函数，名为NewCodecFunc
// 接受一个io.ReadWriteCloser类型的参数，返回一个Codec类型的值。
type NewCodecFunc func(closer io.ReadWriteCloser) Codec

type Type string

//  2 种 Codec，Gob 和 Json
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
