# 📝 GeeRPC

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，允许调用不同进程空间的程序。
RPC 的客户端和服务器可以在一台机器上，也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

GeeRPC实现了Go语言官方的标准库`net/rpc`，并在此基础上，新增了：
1. 协议交换(protocol exchange)
2. 注册中心(registry)、服务发现(service discovery)
3. 负载均衡(load balance)
4. 超时处理(timeout processing)等特性。

最终代码约1000行

## 🚀 快速开始

1. 安装依赖：

```bash
npm install
# 或
yarn install
```

2. 启动开发服务器：

```bash
npm run dev
# 或
yarn dev
```

3. 访问 http://localhost:3000 即可开始使用。

## 📖 使用说明

1. 在左侧输入框中填写项目信息，如标题、介绍、安装方式等。

2. 在右侧选择需要添加的部分，如徽标、截图、代码示例等。

3. 点击「生成 README」按钮，即可自动生成 README 文件。

4. 将生成的 README 复制到项目根目录下的 README.md 文件中即可。

## 💡 功能特性
### 服务端与消息编码
为了实现上更简单，GeeRPC 客户端固定采用 JSON 编码 Option，后续的`header`和`body`的编码方式由`Option`中的`CodeType`指定，
服务端首先使用`JSON`解码`Option`，然后通过`Option`的`CodeType`解码剩余的内容。

```
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
```
可以改进的点：涉及协议协商的这部分信息，可以设计固定的字节来传输的。

在一次连接中，`Option`固定在报文的最开始，`Header` 和 `Body` 可以有多个，即报文可能是这样的。
```
| Option | Header1 | Body1 | Header2 | Body2 | ...
```

服务端工作流程：
1. 首先需要完成一开始的协议交换，即接收`Option`，协商好消息的编解码方式之后，再创建一个子协程调用`serveConn()`处理后续的请求。
2. `serveConn()`方法中，首先需要从连接中读取完整的消息，然后解码`Option`，根据`Option`中的`CodeType`解码`Header`和`Body`，最后根据`Header`中的`ServiceMethod`找到对应的`Service`实例，然后调用`Service`实例的`Call()`方法，将`Body`传入，得到`Call`实例，最后将`Call`实例中的`Reply`编码后发送给客户端。
3. `Call`实例中的`Error`字段不为空，说明调用过程中出现了错误，需要将`Call`实例中的`Error`字段编码后发送给客户端。
4. `Call`实例中的`Error`字段为空，说明调用过程中没有出现错误，需要将`Call`实例中的`Reply`字段编码后发送给客户端。
5. 如果`Call`实例中的`Done`字段不为空，说明调用过程中出现了错误，需要将`Call`实例中的`Error`字段编码后发送给客户端。
6. 如果`Call`实例中的`Done`字段为空，说明调用过程中没有出现错误，需要将`Call`实例中的`Reply`字段编码后发送给客户端。

### 客户端

对 net/rpc 而言，一个函数需要能够被远程调用，需要满足如下五个条件：
1. the method’s type is exported.
2. the method is exported.
3. the method has two arguments, both exported (or builtin) types.
4. the method’s second argument is a pointer.
5. the method has return type error.

更直观一些：
```
func (t *T) MethodName(argType T1, replyType *T2) error
```

Client工作流程：
1. 创建 Client 实例时，首先需要完成一开始的协议交换，即发送`Option`给服务端，协商好消息的编解码方式之后， 再创建一个子协程调用 receive() 接收`Option`响应
2. 调用 Call() 方法时，首先需要创建一个`Call`实例，然后将`Call`实例放入`Client`的`pending`中，然后调用 send() 方法发送请求，最后调用 receive() 方法接收响应。
3. send() 方法中，首先需要将`Call`实例中的`ServiceMethod`、`Seq`、`Error`等信息编码成`Header`，然后将`Header`和`Call`实例中的`Args`编码成`Body`，最后将`Option`、`Header`和`Body`编码成一个完整的消息，发送给服务端。
4. receive() 方法中，首先需要从连接中读取完整的消息，然后解码`Option`，根据`Option`中的`CodeType`解码`Header`和`Body`，最后根据`Header`中的`Seq`找到对应的`Call`实例，将`Body`解码到`Call`实例的`Reply`字段中。
5. 如果`Call`实例中的`Error`字段不为空，说明调用过程中出现了错误，需要返回错误信息。
6. 如果`Call`实例中的`Error`字段为空，说明调用过程中没有出现错误，需要返回`Reply`字段。
7. 如果`Call`实例中的`Done`字段不为空，说明调用过程中出现了错误，需要将`Call`实例中的`Done`字段置为`true`，并调用`Call`实例中的`Done`方法。

## 🤝 贡献指南

如有任何问题或建议，欢迎提交 issue 或 PR。

## 📄 许可协议

GeeRPC 使用 MIT 许可协议。详情请参见 [LICENSE](./LICENSE) 文件。