package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strings"

	"google.golang.org/grpc"

	"golang.org/x/net/context"
	"golang.org/x/net/websocket"

	grpchello "cmux/helloworld"
	"github.com/soheilhy/cmux"
)

type exampleHTTPHandler struct{}

func (h *exampleHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "example http response")
}

func serveHTTP(l net.Listener) {
	s := &http.Server{
		Handler: &exampleHTTPHandler{},
	}
	if err := s.Serve(l); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

func EchoServer(ws *websocket.Conn) {
	if _, err := io.Copy(ws, ws); err != nil {
		panic(err)
	}
}

func serveWS(l net.Listener) {
	s := &http.Server{
		Handler: websocket.Handler(EchoServer),
	}
	if err := s.Serve(l); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

type TestRPCRcvr struct{}

func (r TestRPCRcvr) Test(i int, j *int) error {
	*j = i
	return nil
}

func serveRPC(l net.Listener) {
	s := rpc.NewServer()
	if err := s.Register(TestRPCRcvr{}); err != nil {
		panic(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			if err != cmux.ErrListenerClosed {
				panic(err)
			}
			return
		}
		go s.ServeConn(c)
	}

}

type grpcServer struct{}

func (s *grpcServer) SayHello(ctx context.Context, in *grpchello.HelloRequest) (
	*grpchello.HelloReply, error) {

	return &grpchello.HelloReply{Message: "Hello " + in.Name + " from cmux"}, nil
}

func serveGRPC(l net.Listener) {
	grpcs := grpc.NewServer()
	grpchello.RegisterGreeterServer(grpcs, &grpcServer{})
	if err := grpcs.Serve(l); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Panic(err)
	}
	m := cmux.New(l)

	// We first match the connection against HTTP2 fields. If matched, the
	// connection will be sent through the "grpcl" listener.
	//grpcl := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	// Notice: grpc-go will stop sending RPCs before HTTP/2 SETTINGS frame is received
	grpcl := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	//Otherwise, we match it againts a websocket upgrade request.
	wsl := m.Match(cmux.HTTP1HeaderField("Upgrade", "websocket"))

	// Otherwise, we match it againts HTTP1 methods. If matched,
	// it is sent through the "httpl" listener.
	httpl := m.Match(cmux.HTTP1Fast())
	// If not matched by HTTP, we assume it is an RPC connection.
	rpcl := m.Match(cmux.Any())

	// Then we used the muxed listeners.
	go serveGRPC(grpcl)
	go serveWS(wsl)
	go serveHTTP(httpl)
	go serveRPC(rpcl)

	if err := m.Serve(); !strings.Contains(err.Error(), "use of closed network connection") {
		panic(err)
	}

}
