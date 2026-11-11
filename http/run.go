//go:build !windows

package http

import (
    "context"
    "errors"
    stdhttp "net/http"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
)

func Run(engine *gin.Engine, address string) error {
    listener, err := net.Listen("tcp", address)
    if err != nil { return err }
    server := &stdhttp.Server{Handler: engine, ReadHeaderTimeout: 10 * time.Second}
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    done := make(chan error, 1)
    go func() { done <- server.Serve(listener) }()
    select {
    case err := <-done:
        if errors.Is(err, stdhttp.ErrServerClosed) { return nil }
        return err
    case <-ctx.Done():
        deadline, cancel := context.WithTimeout(context.Background(), 20 * time.Second)
        defer cancel()
        if err := server.Shutdown(deadline); err != nil { _ = server.Close(); return err }
        err := <-done
        if errors.Is(err, stdhttp.ErrServerClosed) { return nil }
        return err
    }
}
