package client

import (
	"errors"
	"fmt"
	"geeRPC/codec/codec"
)

// 接收功能，接收到的响应有三种情况：
// call 不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
// call 存在，但服务端处理出错，即 h.Error 不为空。
// call 存在，服务端处理正常，那么需要从 body 中读取 Reply 的值。
func (client *Client) receive() {
	var err error
	for err == nil {
		var header codec.Header
		if err = client.cc.ReadHeader(&header); err != nil {
			break
		}
		// removeCall 根据 seq，从 client.pending 中移除对应的 call，并返回。
		call := client.removeCall(header.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)

		case header.Error != "":
			call.Error = fmt.Errorf(header.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// error occurs, so terminateCalls pending calls
	client.terminateCalls(err)
}
