package client

// Call 封装结构体 Call 来承载一次 RPC 调用所需要的信息
type Call struct {
	Seq           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // 函数的参数
	Reply         interface{} // 从函数返回
	Error         error
	Done          chan *Call // Strobes when call is complete.
}

// 为了支持异步调用，Call 结构体中添加了一个字段 Done, Done 的类型是 chan *Call，当调用结束时，会调用 call.done() 通知调用方
func (call *Call) done() {
	call.Done <- call
}

// 实现和 Call 相关的三个方法。
// registerCall 将参数 call 添加到 client.pending 中，并更新 client.seq。
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// removeCall 根据 seq，从 client.pending 中移除对应的 call，并返回。
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// terminateCalls 服务端或客户端发生错误时调用，将 shutdown 设置为 true，且将错误信息通知所有 pending 状态的 call。
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}
