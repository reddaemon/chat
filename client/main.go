package main

import (
	"bufio"
	pb "chat/server/pb"
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"time"
)

var (
	serverAddr = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	userName   = flag.String("username", "", "your username in the chat")
)

const (
	defaultTimeout = 10800
)

// TODO
// 1. сделать второй тип сервера и клиента (второй клиент может быть чем угодно, например, grpc)
// 2. сервер живёт в отдельном сервисе
// 3. 2 сервера между собой соединены очередью


func sending(stream pb.ChatManager_ChatClient,
	waitChannel chan struct{}, errorChannel chan error) {
	for {
		reader := bufio.NewReader(os.Stdin)
		message := pb.Message{}
		sender := pb.User{}
		sender.Name = *userName
		fmt.Println("Enter your message: ")
		message.Sender = &sender
		var err error
		message.Body, err = reader.ReadString('\n')
		if err != nil {
			errorChannel <- err
		}
		message.Id = uuid.Must(uuid.NewRandom()).String()
		fmt.Println("Enter recipient name: ")
		recipient := pb.User{}
		recipient.Name, err = reader.ReadString('\n')
		if err != nil {
			errorChannel <- err
		}
		message.Recipient = &recipient
		err = stream.Send(&message)
		if err == io.EOF {
			fmt.Println("EOF")
			errorChannel <- err
			return
		}
		if err != nil {
			return
		}
		log.Printf("sent: %s", &message)
		time.Sleep(1 * time.Second)
		waitChannel <- struct{}{}
	}
}

func reading(stream pb.ChatManager_ChatClient) {
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s: %s", resp.Sender.Name, resp.Body)
	}
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("error due to close connection: %s", err)
		}
	}(conn)

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*defaultTimeout)
	defer cancel()
	client := pb.NewChatManagerClient(conn)
	stream, err := client.Chat(ctxTimeout)
	if err != nil {
		log.Fatal(err)
	}

	waitChannel := make(chan struct{})
	errorChannel := make(chan error)

	go sending(stream, waitChannel, errorChannel)
	go reading(stream)

	for {
		select {
		case err := <-errorChannel:
			if err != nil {
				log.Printf("%s", err.Error())
			}
			return
		case <-waitChannel:
		case <-ctx.Done:
			if ctx.Error != context.Canceled {
				return
			}
		}
	}

}
