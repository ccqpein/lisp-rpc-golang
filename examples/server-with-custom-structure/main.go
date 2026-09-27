package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/server"
)

// SomeData represents a custom RPC request structure.
type SomeData struct {
	Id   int64  `lisp-rpc:"id"`
	Data string `lisp-rpc:"data"`
}

// ToRPCType classifies SomeData as an RPC command named "some-data".
func (SomeData) ToRPCType() server.RPCType {
	return server.NewRPCTypeRPC("some-data")
}

// SomeResp represents the typed response structure.
type SomeResp struct {
	Message string `lisp-rpc:"message"`
	Result  int64  `lisp-rpc:"result"`
}

func main() {
	// 1. Initialize the Lisp-RPC Server engine
	srv := server.New()

	// 2. Register typed RPC handler
	_, err := srv.Register(func(ctx context.Context, req SomeData) (SomeResp, error) {
		fmt.Printf("[Server] Received request: id=%d, data=%s\n", req.Id, req.Data)
		return SomeResp{
			Message: fmt.Sprintf("Processed data: %s", req.Data),
			Result:  req.Id * 10,
		}, nil
	})
	if err != nil {
		log.Fatalf("Handler registration error: %v", err)
	}

	// 3. Demo dispatch directly using Handle()
	rawReq := `(some-data :id 42 :data "Hello Go RPC")`
	fmt.Printf("[Client] Sending direct payload: %s\n", rawReq)

	respStr, err := srv.Handle(rawReq)
	if err != nil {
		log.Fatalf("Handle error: %v", err)
	}
	fmt.Printf("[Client] Received serialized response:\n%s\n\n", respStr)

	// 4. Native net/http integration (srv implements http.Handler)
	httpServer := httptest.NewServer(srv)
	defer httpServer.Close()

	fmt.Printf("[HTTP] Testing POST to %s\n", httpServer.URL)
	resp, err := http.Post(httpServer.URL, "application/x-lisp-rpc", strings.NewReader(rawReq))
	if err != nil {
		log.Fatalf("HTTP POST error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[HTTP] Status: %s\n", resp.Status)
}
