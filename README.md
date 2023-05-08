# 📝 GeeRPC

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，允许调用不同进程空间的程序。
RPC 的客户端和服务器可以在一台机器上，也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

GeeRPC实现了Go语言官方的标准库`net/rpc`，并在此基础上，新增了：
1. 协议交换(protocol exchange)
2. 注册中心(registry)
3. 服务发现(service discovery)
4. 负载均衡(load balance)
5. 超时处理(timeout processing)等特性。

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

### 服务注册
1. 通过反射实现服务注册功能
2. 在服务端实现服务调用

#### 结构体映射为服务
RPC 框架的一个基础能力是：像调用本地程序一样调用远程服务。那如何将程序映射为服务呢?  
对 Go 来说，这个问题就变成了如何将结构体的方法映射为服务。

对 net/rpc 而言，一个函数需要能够被远程调用，需要满足如下五个条件：  
the method’s type is exported. – 方法所属类型是导出的。  
the method is exported. – 方式是导出的。  
the method has two arguments, both exported (or builtin) types. – 两个入参，均为导出或内置类型。  
the method’s second argument is a pointer. – 第二个入参必须是一个指针。  
the method has return type error. – 返回值为 error 类型。  

更直观一些：
```go
func (t *T) MethodName(argType T1, replyType *T2) error
```

假设客户端发过来一个请求，包含 ServiceMethod 和 Argv。
```json
{
    "ServiceMethod":"T.MethodName",
    "Argv":"0101110101..." // 序列化之后的字节流
}
```
通过 “T.MethodName” 可以确定调用的是类型 T 的 MethodName，如果硬编码实现这个功能，那么就要写多个`switch-case`覆盖所有情况，每个情况要编写等量的代码。
但是如果借助反射就可以将这个映射过程自动化，可以非常容易地获取某个结构体的所有方法，并且能够通过方法，获取到该方法所有的参数类型与返回值。

### 超时处理机制
超时处理是 RPC 框架一个比较基本的能力，如果缺少超时处理机制，无论是服务端还是客户端都容易因为网络或其他错误导致挂死，资源耗尽，这些问题的出现大大地降低了服务的可用性。

纵观整个远程调用的过程，需要客户端处理超时的地方有：
1. 与服务端建立连接，导致的超时
2. 发送请求到服务端，写报文导致的超时
3. 等待服务端处理时，等待处理导致的超时（比如服务端已挂死，迟迟不响应）
4. 从服务端接收响应时，读报文导致的超时

需要服务端处理超时的地方有： 
1. 读取客户端请求报文时，读报文导致的超时
2. 发送响应报文时，写报文导致的超时
3. 调用映射服务的方法时，处理报文导致的超时

GeeRPC 在 3 个地方添加了超时处理机制。分别是：
1. 客户端创建连接时
2. 客户端 Client.Call() 整个过程导致的超时（包含发送报文，等待处理，接收报文所有阶段）
3. 服务端处理报文，即 Server.handleRequest 超时。  

### 支持HTTP协议
RPC 的消息格式与标准的 HTTP 协议并不兼容，在这种情况下，就需要一个协议的转换过程。HTTP 协议的 CONNECT 方法恰好提供了这个能力，CONNECT 一般用于代理服务。

浏览器与服务器之间的 HTTPS 通信都是加密的，浏览器通过代理服务器发起 HTTPS 请求时，由于请求的站点地址和端口号都是加密保存在 HTTPS 请求报文头中，代理服务器无法获知目标服务器ip和端口

为了解决这个问题，浏览器通过 HTTP 明文形式向代理服务器发送一个 CONNECT 请求告诉代理服务器目标地址和端口，代理服务器接收到这个请求后，会在对应端口与目标站点建立一个 TCP 连接，连接建立成功后返回 HTTP 200 状态码告诉浏览器与该站点的加密通道已经完成。
接下来代理服务器仅需透传浏览器和服务器之间的加密数据包即可，代理服务器无需解析 HTTPS 报文。

举一个简单例子：

浏览器向代理服务器发送 CONNECT 请求。
```http
CONNECT geektutu.com:443 HTTP/1.0
```
代理服务器返回 HTTP 200 状态码表示连接已经建立。
```http request
HTTP/1.0 200 Connection Established
```
之后浏览器和服务器开始 HTTPS 握手并交换加密数据，代理服务器只负责传输彼此的数据包，并不能读取具体数据内容（代理服务器也可以选择安装可信根证书解密 HTTPS 报文）。
事实上，这个过程其实是通过代理服务器将 HTTP 协议转换为 HTTPS 协议的过程。对 RPC 服务端来，需要做的是将 HTTP 协议转换为 RPC 协议，对客户端来说，需要新增通过 HTTP CONNECT 请求创建连接的逻辑。

服务端支持 HTTP 协议
那通信过程应该是这样的：

客户端向 RPC 服务器发送 CONNECT 请求
```http request
CONNECT 10.0.0.1:9999/_geerpc_ HTTP/1.0
```
RPC 服务器返回 HTTP 200 状态码表示连接建立。
```http request
HTTP/1.0 200 Connected to Gee RPC
```
客户端使用创建好的连接发送 RPC 报文，先发送 Option，再发送 N 个请求报文，服务端处理 RPC 请求并响应。

### 负载均衡
应用场景：有多个服务实例，每个实例提供相同的功能，为了提高整个系统的吞吐量，每个实例部署在不同的机器上。客户端可以选择任意一个实例进行调用。

对于 RPC 框架来说，我们可以很容易地想到这么几种策略：
1. 随机选择策略 - 从服务列表中随机选择一个。
2. 轮询算法(Round Robin) - 依次调度不同的服务器，每次调度执行 i = (i + 1) mode n。
3. 加权轮询(Weight Round Robin) - 在轮询算法的基础上，为每个服务实例设置一个权重，高性能的机器赋予更高的权重，也可以根据服务实例的当前的负载情况做动态的调整，例如考虑最近5分钟部署服务器的 CPU、内存消耗情况。
4. 哈希/一致性哈希策略 - 依据请求的某些特征，计算一个 hash 值，根据 hash 值将请求发送到对应的机器。一致性 hash 还可以解决服务实例动态添加情况下，调度抖动的问题。一致性哈希的一个典型应用场景是分布式缓存服务。

#### 服务发现
负载均衡的前提是有多个服务实例，代码中实现了一个最基础的服务发现模块 Discovery。为了与通信部分解耦，这部分的代码统一放置在 xclient 子目录下。

定义 2 个类型：
1. SelectMode 代表不同的负载均衡策略，简单起见，GeeRPC 仅实现 Random 和 RoundRobin 两种策略。
2. Discovery 是一个接口类型，包含了服务发现所需要的最基本的接口。 
   1. Refresh() 从注册中心更新服务列表 
   2. Update(servers []string) 手动更新服务列表 
   3. Get(mode SelectMode) 根据负载均衡策略，选择一个服务实例 
   4. GetAll() 返回所有的服务实例

### 服务发现和注册中心
实现一个简单的注册中心，支持服务注册、接收心跳等功能
客户端实现基于注册中心的服务发现机制，代码约 250 行

#### 注册中心的位置
<img src="https://image-20220620.oss-cn-guangzhou.aliyuncs.com/image/20230508162711.png" alt="image-20230505220714472" style="zoom:30%;" />

注册中心的位置如上图所示。注册中心的好处在于，客户端和服务端都只需要感知注册中心的存在，而无需感知对方的存在。更具体一些：
1. 服务端启动后，向注册中心发送注册消息，注册中心得知该服务已经启动，处于可用状态。一般来说，服务端还需要定期向注册中心发送心跳，证明自己还活着。
2. 客户端向注册中心询问，当前哪天服务是可用的，注册中心将可用的服务列表返回客户端。
3. 客户端根据注册中心得到的服务列表，选择其中一个发起调用。

如果没有注册中心，客户端需要硬编码服务端的地址，而且没有机制保证服务端是否处于可用状态。当然注册中心的功能还有很多，比如配置的动态同步、通知机制等。比较常用的注册中心有 etcd、zookeeper、consul，一般比较出名的微服务或者 RPC 框架，这些主流的注册中心都是支持的。

## 更多
本项目参照 golang 标准库 net/rpc，实现了服务端以及支持并发的客户端，并且支持选择不同的序列化与反序列化方式；为了防止服务挂死，在其中一些关键部分添加了超时处理机制；支持 TCP、Unix、HTTP 等多种传输协议；支持多种负载均衡模式，最后还实现了一个简易的服务注册和发现中心。

实际应用时，需要额外实现一个程序去调用服务。
在多台机器上，main函数中启动 RPC 服务，配置好注册中心，在不同的机器上运行 main 函数，就可以实现分布式的 RPC 服务了。

## 🤝 贡献指南

如有任何问题或建议，欢迎提交 issue 或 PR。

## 📄 许可协议

GeeRPC 使用 MIT 许可协议。详情请参见 [LICENSE](./LICENSE) 文件。