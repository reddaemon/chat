package main

import (
	"bufio"
	pb "chat/server/pb"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"time"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.example.com", "The server name used to verify the hostname returned by the TLS handshake")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = "ca_cert.pem"
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

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
	ctx := context.Background()
	client := pb.NewChatManagerClient(conn)
	stream, err := client.Chat(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Enter your name: ")
			message := pb.Message{}
			sender := pb.User{}
			sender.Name, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			fmt.Println("Enter your message: ")
			message.Sender = &sender
			message.Body, err = reader.ReadString('\n')
			message.Id = uint64(time.Now().UnixNano())
			fmt.Println("Enter recipient name: ")
			recipient := pb.User{}
			recipient.Name, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			message.Recipient = &recipient
			err = stream.Send(&message)
			if err == io.EOF {
				fmt.Println("EOF")
				return
			}
			if err != nil {
				return
			}
			log.Printf("sent: %s", &message)
			time.Sleep(1 * time.Second)
		}
	}()
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s: %s", resp.Sender.Name, resp.Body)
	}

}
