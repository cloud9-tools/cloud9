package server

import (
	"net"
	"time"
)

type KeepAliveListener struct{ L net.Listener }

func (l KeepAliveListener) Accept() (net.Conn, error) {
	c, err := l.L.Accept()
	if err != nil {
		if tc, ok := c.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(1 * time.Minute)
		}
	}
	return c, err
}

func (l KeepAliveListener) Close() error {
	return l.L.Close()
}

func (l KeepAliveListener) Addr() net.Addr {
	return l.L.Addr()
}

var _ net.Listener = (*KeepAliveListener)(nil)
