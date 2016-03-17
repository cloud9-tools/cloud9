package server

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud9-tools/cloud9/repo"
)

type CloudServer struct {
	Repo   *repo.Repo
	stopch chan struct{}
}

func New(dir string) (*CloudServer, error) {
	r, err := repo.Open(dir)
	if err != nil {
		return nil, err
	}
	return &CloudServer{
		Repo:   r,
		stopch: make(chan struct{}),
	}, nil
}

func (srv *CloudServer) ListenAndServe(proto, laddr string) error {
	if laddr == "" {
		laddr = ":http"
	}
	l, err := net.Listen(proto, laddr)
	if err != nil {
		return err
	}
	log.Printf("listening on %v", l.Addr())
	return srv.Serve(l)
}

func (srv *CloudServer) Serve(l net.Listener) error {
	mux := http.NewServeMux()
	for _, h := range StaticHandlers {
		mux.Handle(h.Path, h)
	}
	mux.Handle("/", HomeHandler{})
	userHandler := &UserHandler{srv.Repo}
	mux.Handle("/user", userHandler)
	mux.Handle("/user/", userHandler)
	groupHandler := &GroupHandler{srv.Repo}
	mux.Handle("/group", groupHandler)
	mux.Handle("/group/", groupHandler)
	blobHandler := &BlobHandler{srv.Repo}
	mux.Handle("/blob", blobHandler)
	mux.Handle("/blob/", blobHandler)

	httpserver := &http.Server{
		Addr:         l.Addr().String(),
		Handler:      UnproxyHandler{mux},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigch)
	go (func() {
		select {
		case sig := <-sigch:
			log.Printf("got signal %v", sig)
		case <-srv.stopch:
		}
		l.Close()
	})()

	err := httpserver.Serve(KeepAliveListener{L: l})
	log.Printf("graceful shutdown")
	return err
}

func (srv *CloudServer) Stop() {
	srv.stopch <- struct{}{}
}

func (srv *CloudServer) Close() error {
	return srv.Repo.Close()
}
