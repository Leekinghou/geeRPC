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
- 支持自定义项目信息，如标题、介绍、安装方式等。

- 支持添加徽标、截图、代码示例等常用模块。

- 支持自动生成 TOC（Table of Contents）目录。

- 支持多种风格的 README 模板，可根据需求进行选择。

## 🤝 贡献指南

如有任何问题或建议，欢迎提交 issue 或 PR。

## 📄 许可协议

README GPT 使用 MIT 许可协议。详情请参见 [LICENSE](./LICENSE) 文件。