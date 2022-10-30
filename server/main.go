package main

import (
	pb "chat/server/pb"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type chatServer struct {
	pb.UnimplementedChatManagerServer
	mu       *sync.RWMutex
	messages map[uint64][]*pb.Message
	users    map[string]pb.ChatManager_ChatServer
}

const (
	defaultPort = "50051"
)

func (s *chatServer) Chat(stream pb.ChatManager_ChatServer) error {
	uid := uuid.Must(uuid.NewRandom()).String()

	s.addUser(uid, stream)
	defer s.removeUser(uid)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic: %v", err)
			os.Exit(1)
		}
	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			continue
		}

		if err != nil {
			return err
		}

		log.Printf("Got message id %s, from  %s to %s", in.Id, in.Sender.Name, in.Recipient.Name)

		log.Printf("broadcast: %s", in.Body)
		for _, ss := range s.getUsers() {
			if err := ss.Send(in); err != nil {
				log.Printf("broadcast err: %v", err)
			}
		}
	}
}

func newChatServer() *chatServer {
	s := &chatServer{
		messages: make(map[uint64][]*pb.Message),
		users:    make(map[string]pb.ChatManager_ChatServer),
		mu:       &sync.RWMutex{},
	}
	return s
}

func (s *chatServer) addUser(uid string, user pb.ChatManager_ChatServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[uid] = user
}

func (s *chatServer) removeUser(uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, uid)
}

func (s *chatServer) getUsers() []pb.ChatManager_ChatServer {
	var users []pb.ChatManager_ChatServer

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterChatManagerServer(grpcServer, newChatServer())
	log.Printf("listen on port %s", port)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}

}
