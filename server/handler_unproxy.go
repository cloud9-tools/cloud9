package server

import (
	"net"
	"net/http"
	"strings"
)

type UnproxyHandler struct{ H http.Handler }

func (handler UnproxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse this more thoroughly
	if xff, ok := r.Header[XForwardedFor]; ok {
		_, port, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			port = "0"
		}
		xff = strings.Split(strings.Join(xff, ","), ",")
		host := strings.Trim(xff[len(xff)-1], " \t")
		r.RemoteAddr = net.JoinHostPort(host, port)
	}
	handler.H.ServeHTTP(w, r)
}
