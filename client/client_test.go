package client

import (
	"geeRPC/service"
	"net"
	"os"
	"runtime"
	"testing"
)

func TestXDial(t *testing.T) {
	if runtime.GOOS == "linux" {
		ch := make(chan struct{})
		addr := "/tmp/geerpc.sock"
		go func() {
			_ = os.Remove(addr)
			l, err := net.Listen("unix", addr)
			if err != nil {
				t.Fatal("failed to listen unix socket")
			}
			ch <- struct{}{}
			service.Accept(l)
		}()
		<-ch
		_, err := XDial("unix@" + addr)
		//_assert(err == nil, "failed to connect unix socket")
		if err != nil {
			t.Fatal("failed to connect unix socket")
		}
	}
}
