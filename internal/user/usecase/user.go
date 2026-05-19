package usecase
import (
	"context"
	"banking-service/internal/user/domain"
)
type userUseCase struct{}
func NewUserUseCase() domain.UserUseCase { return &userUseCase{} }
func (u *userUseCase) RegisterUser(ctx context.Context, email, fullName string) (string, error) { return "user-123", nil }
func (u *userUseCase) GetUserProfile(ctx context.Context, userID string) (string, string, string, error) { return userID, "test@test.com", "Test User", nil }
