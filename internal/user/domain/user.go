package domain
import "context"
type UserUseCase interface {
	RegisterUser(ctx context.Context, email, fullName string) (string, error)
	GetUserProfile(ctx context.Context, userID string) (string, string, string, error)
}
