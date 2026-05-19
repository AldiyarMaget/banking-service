package grpc
import (
	"context"
	"banking-service/internal/user/domain"
	userv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/user/v1"
)
type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	usecase domain.UserUseCase
}
func NewUserHandler(u domain.UserUseCase) *UserHandler { return &UserHandler{usecase: u} }
func (h *UserHandler) RegisterUser(ctx context.Context, req *userv1.RegisterUserRequest) (*userv1.RegisterUserResponse, error) {
	id, _ := h.usecase.RegisterUser(ctx, req.Email, req.FullName)
	return &userv1.RegisterUserResponse{UserId: id}, nil
}
func (h *UserHandler) GetUserProfile(ctx context.Context, req *userv1.GetUserProfileRequest) (*userv1.GetUserProfileResponse, error) {
	id, email, name, _ := h.usecase.GetUserProfile(ctx, req.UserId)
	return &userv1.GetUserProfileResponse{UserId: id, Email: email, FullName: name}, nil
}
