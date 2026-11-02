package httpserver

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestRunStopsOnContextCancel(t *testing.T) {
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, &http.Server{Addr: addr, Handler: http.NotFoundHandler()}) }()

	// Wait until it is serving.
	for i := 0; ; i++ {
		if resp, err := http.Get("http://" + addr); err == nil {
			resp.Body.Close()
			break
		}
		if i > 100 {
			t.Fatal("server never came up")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("clean shutdown returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRunReturnsServeError(t *testing.T) {
	// Two servers on the same address: the second cannot bind.
	addr := freeAddr(t)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	err = Run(context.Background(), &http.Server{Addr: addr, Handler: http.NotFoundHandler()})
	if err == nil {
		t.Fatal("want bind error, got nil")
	}
}
