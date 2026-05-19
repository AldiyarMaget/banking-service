package main
import (
	"log"
	"net"
	"banking-service/internal/analytics/delivery/grpc"
	"banking-service/internal/analytics/usecase"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
	googlegrpc "google.golang.org/grpc"
)
func main() {
	uc := usecase.NewAnalyticsUseCase()
	handler := grpc.NewAnalyticsHandler(uc)
	listener, _ := net.Listen("tcp", ":50054")
	grpcServer := googlegrpc.NewServer()
	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, handler)
	log.Println("Analytics Service running on :50054")
	grpcServer.Serve(listener)
}
