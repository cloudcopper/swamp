package infra

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/cloudcopper/swamp/ports"
)

type WebServer struct {
	log     ports.Logger
	srv     *http.Server
	closeWg sync.WaitGroup
}

func NewWebServer(log ports.Logger, addr string, handler http.Handler) (*WebServer, error) {
	log = log.With(slog.String("entity", "WebServer"))

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &WebServer{
		log: log,
		srv: &http.Server{Handler: handler},
	}

	s.closeWg.Add(1)
	go func() {
		defer s.closeWg.Done()
		log.Info("started", slog.String("addr", addr))
		defer log.Warn("complete")
		defer l.Close()
		err := s.srv.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			log.Error("serve error", slog.Any("err", err))
		}
	}()

	return s, nil
}

func (s *WebServer) Close() {
	s.log.Info("closing")
	if err := s.srv.Shutdown(context.TODO()); err != nil {
		s.log.Error("shutdown error", slog.Any("err", err))
	}
	s.closeWg.Wait()
	s.srv = nil
}
