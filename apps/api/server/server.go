package server

import (
	"commerce/api/configs"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	l      slog.Logger
	router *gin.Engine
	config *configs.Config
}

func NewServer(l slog.Logger, router *gin.Engine, config *configs.Config) *Server {
	return &Server{l: l, router: router, config: config}
}

func (s *Server) Run() {
	srv := &http.Server{
		Addr:    s.config.Server.Address,
		Handler: s.router.Handler(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.l.Error(err.Error())
		}
	}()
	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 30 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.l.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		s.l.Error(err.Error())
	}
	s.l.Info("Server exiting")
}
