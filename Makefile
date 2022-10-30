.PHONY: compile

compile:
	 protoc --go_out=server/pb --go_opt=paths=source_relative --go-grpc_out=server/pb --go-grpc_opt=paths=source_relative chat.proto

build-server:
	go build -o chat-server ./server/main.go

build-client:
	go build -o chat-client ./client/main.go
