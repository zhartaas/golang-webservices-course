package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type Server struct {
	ACLData map[string]interface{}
	UnimplementedBizServer
	UnimplementedAdminServer
	LogChannel  map[chan Event]struct{}
	mu          sync.Mutex
	RemoteAddr  string
	StatChannel map[chan Event]struct{}
}

func NewServer(acl map[string]interface{}, addr string) *Server {
	return &Server{ACLData: acl, RemoteAddr: addr}
}

func (s *Server) aclStreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	//fmt.Println(info.FullMethod)
	md, ok := metadata.FromIncomingContext(ss.Context())
	//fmt.Println(md)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "missing metadata")
	}
	consumerName := md["consumer"]
	if len(consumerName) == 0 {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	availableMethods, ok := s.ACLData[consumerName[0]].([]interface{})
	if !ok {
		return status.Error(codes.Unauthenticated, "unknown user")
	}
	ok = false
	for _, method := range availableMethods {
		if method.(string) == info.FullMethod {
			ok = true
		}
	}

	if !ok {
		return status.Errorf(codes.Unauthenticated, "method not available for this consumder")
	}
	host := s.RemoteAddr + info.FullMethod
	evt := Event{Timestamp: time.Now().Unix(), Consumer: consumerName[0], Method: info.FullMethod, Host: host}

	if s.LogChannel != nil {
		for ch := range s.LogChannel {
			ch <- evt
		}
	}
	if s.StatChannel != nil {
		for ch := range s.StatChannel {
			ch <- evt
		}
	}

	return handler(srv, ss)
}

func (srv *Server) aclInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	//fmt.Println(info.FullMethod)
	md, ok := metadata.FromIncomingContext(ctx)
	//fmt.Println(md)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	consumerName := md["consumer"]
	if len(consumerName) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	availableMethods, ok := srv.ACLData[consumerName[0]].([]interface{})

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown user")
	}
	ok = false
	if consumerName[0] == "biz_admin" {
		if strings.HasPrefix(info.FullMethod, "/main.Biz/") {
			ok = true
		}
	} else {
		for _, method := range availableMethods {
			if method.(string) == info.FullMethod {
				ok = true
			}
		}
	}
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "method not available for this consumer")
	}
	host := srv.RemoteAddr + info.FullMethod
	evt := Event{Timestamp: time.Now().Unix(), Consumer: consumerName[0], Method: info.FullMethod, Host: host}
	if srv.LogChannel != nil {
		for ch := range srv.LogChannel {
			ch <- evt
		}
	}
	if srv.StatChannel != nil {
		for ch := range srv.StatChannel {
			ch <- evt
		}
	}

	return handler(ctx, req)
}

func StartMyMicroservice(ctx context.Context, listenAddr, ACLData string) error {
	acl := make(map[string]interface{})
	if err := json.Unmarshal([]byte(ACLData), &acl); err != nil {
		// log error
		//fmt.Println(ACLData)
		fmt.Println(err)
		return err
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		// log error
		fmt.Println(err)
		return err
	}
	//fmt.Println("net.Listen")

	//fmt.Println(acl)

	srv := NewServer(acl, listenAddr)
	// add UnaryInterceptor with logging and statistics
	server := grpc.NewServer(
		grpc.UnaryInterceptor(srv.aclInterceptor),
		grpc.StreamInterceptor(srv.aclStreamInterceptor),
	)
	//fmt.Println("Registering server")
	RegisterBizServer(server, srv)
	RegisterAdminServer(server, srv)

	// registering server logs

	//
	//fmt.Println("Start serve...")
	go func() {
		server.Serve(lis)
	}()
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			server.Stop()
		}
	}(ctx)
	return nil
}

func (srv *Server) Check(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (srv *Server) Add(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (srv *Server) Test(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (srv *Server) Logging(in *Nothing, server Admin_LoggingServer) error {
	if srv.LogChannel == nil {
		srv.LogChannel = make(map[chan Event]struct{})
	}
	ch := make(chan Event)
	srv.mu.Lock()
	srv.LogChannel[ch] = struct{}{}
	srv.mu.Unlock()

	defer func() {
		srv.mu.Lock()
		delete(srv.LogChannel, ch)
		srv.mu.Unlock()
		close(ch)
	}()
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				log.Println("Channel closed")
				return nil
			}
			if err := server.Send(&evt); err != nil {
				fmt.Println(&evt)
				log.Println(err)
				return status.Errorf(codes.Internal, "sending error")
			}
		case <-server.Context().Done():
			return server.Context().Err()
		}

	}
	return nil
}

func (srv *Server) Statistics(in *StatInterval, stream Admin_StatisticsServer) error {
	if srv.StatChannel == nil {
		srv.StatChannel = make(map[chan Event]struct{})
	}
	ch := make(chan Event)
	srv.mu.Lock()
	srv.StatChannel[ch] = struct{}{}
	srv.mu.Unlock()

	defer func() {
		srv.mu.Lock()
		delete(srv.StatChannel, ch)
		srv.mu.Unlock()
		close(ch)
	}()

	mu := &sync.Mutex{}
	byMethod := make(map[string]uint64)
	byConsumer := make(map[string]uint64)

	go func() {
		for {
			time.Sleep(time.Second * time.Duration(in.IntervalSeconds))
			stat := Stat{Timestamp: time.Now().Unix(), ByMethod: byMethod, ByConsumer: byConsumer}
			if err := stream.Send(&stat); err != nil {
				log.Println(err)
				return
			}
			byMethod = make(map[string]uint64)
			byConsumer = make(map[string]uint64)
		}
	}()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				log.Println("Channel closed")
				return nil
			}
			mu.Lock()
			byMethod[evt.Method]++
			byConsumer[evt.Consumer]++
			mu.Unlock()
		case <-stream.Context().Done():
			return stream.Context().Err()
		}

	}

	return nil
}
