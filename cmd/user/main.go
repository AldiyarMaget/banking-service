package main
import (
	"log"
	"net"
	"banking-service/internal/user/delivery/grpc"
	"banking-service/internal/user/usecase"
	userv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/user/v1"
	googlegrpc "google.golang.org/grpc"
)
func main() {
	uc := usecase.NewUserUseCase()
	handler := grpc.NewUserHandler(uc)
	listener, _ := net.Listen("tcp", ":50053")
	grpcServer := googlegrpc.NewServer()
	userv1.RegisterUserServiceServer(grpcServer, handler)
	log.Println("User Service running on :50053")
	grpcServer.Serve(listener)
}
