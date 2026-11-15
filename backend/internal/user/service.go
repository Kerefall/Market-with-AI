package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"shopai/internal/auth"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type Service struct {
	repo      *Repository
	jwtSecret string
	tokenTTL  time.Duration
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, tokenTTL: 24 * time.Hour}
}

// Register создаёт аккаунт селлера. Админов публично создать нельзя,
// покупатели работают без аккаунта (контакты хранятся в заказе).
func (s *Service) Register(ctx context.Context, email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, errors.Join(ErrInvalidInput, errors.New("bad email"))
	}
	if len(password) < 8 {
		return nil, errors.Join(ErrInvalidInput, errors.New("password must be at least 8 characters"))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, email, string(hash), RoleSeller)
}

func (s *Service) Login(ctx context.Context, email, password string) (string, *User, error) {
	u, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}
	token, err := auth.Sign(s.jwtSecret, u.ID, u.Role, s.tokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (s *Service) Me(ctx context.Context, id int64) (*User, error) { return s.repo.GetByID(ctx, id) }

func (s *Service) List(ctx context.Context, limit, offset int) ([]User, error) {
	return s.repo.List(ctx, limit, offset)
}
