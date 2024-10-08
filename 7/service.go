package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type Biz struct {
	// Empty
}

// mustEmbedUnimplementedBizServer implements BizServer.
func (biz *Biz) mustEmbedUnimplementedBizServer() {
	panic("unimplemented")
}

func (biz *Biz) Add(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}

func (biz *Biz) Check(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}

func (biz *Biz) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}

type Admin struct {
	Roles         map[string][]string
	Mu            sync.Mutex
	EventChannels map[string]chan *Event

	Stats map[string]*Stat
}

func (admin *Admin) mustEmbedUnimplementedAdminServer() {
	panic("unimplemented")
}

func newAdmin() *Admin {
	return &Admin{}
}

func (admin *Admin) Logging(nothing *Nothing, stream Admin_LoggingServer) error {
	consumer, err := getConsumer(stream.Context())
	if err != nil {
		return err
	}
	eventChannel := make(chan *Event, 20)
	admin.Mu.Lock()
	admin.EventChannels[consumer] = eventChannel
	admin.Mu.Unlock()
	for event := range eventChannel {
		err := stream.Context().Err()
		if err != nil {
			break
		}
		if err := stream.Send(event); err != nil {
			break
		}
	}
	close(eventChannel)
	admin.Mu.Lock()
	delete(admin.EventChannels, consumer)
	admin.Mu.Unlock()
	return nil
}

func (admin *Admin) Statistics(statInterval *StatInterval, stream Admin_StatisticsServer) error {
	currentStat := &Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}
	consumer, err := getConsumer(stream.Context())
	if err != nil {
		return nil
	}
	admin.Mu.Lock()
	admin.Stats[consumer] = currentStat
	admin.Mu.Unlock()

	ctx := stream.Context()

	ticker := time.NewTicker(time.Duration(statInterval.IntervalSeconds) * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			admin.Mu.Lock()
			delete(admin.Stats, consumer)
			admin.Mu.Unlock()
			break

		case <-ticker.C:
			admin.Mu.Lock()
			if err := stream.Send(admin.Stats[consumer]); err != nil {
				return err
			}

			admin.Stats[consumer] = &Stat{
				ByMethod:   make(map[string]uint64),
				ByConsumer: make(map[string]uint64),
			}

			admin.Mu.Unlock()
		}
	}
}

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	roles := make(map[string][]string)
	err := json.Unmarshal([]byte(ACLData), &roles)

	if err != nil {
		return err
	}
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	channels := make(map[string]chan *Event)
	stats := make(map[string]*Stat)

	admin := &Admin{
		Roles:         roles,
		Mu:            sync.Mutex{},
		EventChannels: channels,
		Stats:         stats,
	}
	biz := &Biz{}

	server := grpc.NewServer(
		grpc.StreamInterceptor(getStreamInterceptor(admin)),
		grpc.UnaryInterceptor(getUnaryInterceptor(admin)))

	RegisterBizServer(server, biz)
	RegisterAdminServer(server, admin)
	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.Stop()
		lis.Close()
		return
	}()

	return nil
}

func getStreamInterceptor(admin *Admin) func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		consumer, err := getConsumer(ss.Context())

		if err != nil {
			return err
		}
		p, ok := peer.FromContext(ss.Context())
		if !ok {
			return fmt.Errorf("Can't access context")
		}

		admin.Mu.Lock()
		for _, channel := range admin.EventChannels {
			channel <- &Event{Host: p.Addr.String(), Consumer: consumer, Method: info.FullMethod}
		}
		admin.Mu.Unlock()

		allowedMethods, ok := admin.Roles[consumer]
		if !ok {
			return status.Error(codes.Unauthenticated, "Incorrect role")
		}

		addStat(admin.Stats, consumer, info.FullMethod)

		handler(srv, ss)
		return authenticateRole(info.FullMethod, allowedMethods)
	}
}

func authenticateRole(method string, allowedMethods []string) error {
	hasRights := hasRights(allowedMethods, method)

	if !hasRights {
		return status.Error(codes.Unauthenticated, "Role has no rights")
	}
	return nil
}

func getUnaryInterceptor(admin *Admin) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// HERE
		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("Could not access context")
		}
		consumer, err := getConsumer(ctx)
		if err != nil {
			return nil, err
		}

		admin.Mu.Lock()
		for _, channel := range admin.EventChannels {
			channel <- &Event{Host: p.Addr.String(), Consumer: consumer, Method: info.FullMethod}
		}

		addStat(admin.Stats, consumer, info.FullMethod)

		admin.Mu.Unlock()

		allowedMethods, ok := admin.Roles[consumer]
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Incorrect role")
		}
		err = authenticateRole(info.FullMethod, allowedMethods)

		if err != nil {
			return nil, err
		}
		reply, err := handler(ctx, req)
		return reply, err

	}
}

func hasRights(allowedMethods []string, method string) bool {
	for _, allowedMethod := range allowedMethods {
		if pathsMatch(allowedMethod, method) {
			return true
		}
	}
	return false
}

func pathsMatch(role string, path string) bool {
	roleSlice := strings.Split(role, "/")
	pathSlice := strings.Split(path, "/")

	for i := 0; i < len(pathSlice); i++ {
		pathPiece := pathSlice[i]
		rolePiece := roleSlice[i]
		if rolePiece == "*" {
			return true
		}
		if rolePiece != pathPiece {
			return false
		}
	}
	return true
}

func getConsumer(context context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(context)
	consumers := md.Get("consumer")
	if len(consumers) == 0 {
		return "", status.Error(codes.Unauthenticated, "Role is missing")
	}
	consumer := consumers[0]
	return consumer, nil
}

func addStat(stats map[string]*Stat, consumer string, method string) {
	for _, stat := range stats {
		// By consumer
		val := stat.ByConsumer
		_, ok := val[consumer]
		if !ok {
			val[consumer] = 1
		} else {
			val[consumer] += 1
		}
		// By method
		_, ok = stat.ByMethod[method]
		if !ok {
			stat.ByMethod[method] = 1
		} else {
			stat.ByMethod[method] += 1
		}
	}
}
