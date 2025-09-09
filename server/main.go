package main

import (
	"log"
	"net"

	pb "grpcTages/pbs" // Импорт сгенерированного кода

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	fileService := NewFileServiceServer("uploads")
	pb.RegisterFileServiceServer(server, fileService)

	log.Println("Server started on port 50051")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
