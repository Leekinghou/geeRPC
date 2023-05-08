package client

import (
	"context"
	"errors"
	"fmt"
	"geeRPC/service"
	"net"
	"time"
)

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *service.Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*service.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)

	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()

	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*service.Option) (*Client, error) {
	return dialTimeout(NewClient, network, address, opts...)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// Client.Call 的超时处理机制，使用 context 包实现，控制权交给用户，控制更为灵活。
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}
