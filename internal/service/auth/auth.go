package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

var CredentialError = errors.New("Provided credentials are invalid")
var TokenGenerationError = errors.New("Error while generating token")
var NoValidSessionError = errors.New("No valid session with provided Token")

type AuthService struct {
	storage  storage.Storage
	sessions map[string]session
}

func NewAuthService(storage storage.Storage) *AuthService {
	return &AuthService{
		storage:  storage,
		sessions: make(map[string]session),
	}
}

type session struct {
	user       entity.User
	expiration time.Time
}

func (s session) isExpired() bool {
	return s.expiration.Before(time.Now())
}

func hashPassword(password string) ([]byte, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return bytes, err
}

func checkPasswordHash(password string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

func generateSessionToken() (string, error) {
	token := make([]byte, 64)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(token), nil
}

func (s *AuthService) LoginUser(ctx context.Context, username, password string) (string, *time.Time, error) {
	user, err := s.storage.GetUser(ctx, username)
	if err != nil {
		return "", nil, CredentialError
	}
	if !checkPasswordHash(password, []byte(user.PasswordHash)) {
		return "", nil, CredentialError
	}
	sessionToken, err := generateSessionToken()
	if err != nil {
		return "", nil, TokenGenerationError
	}
	expiration := time.Now().Add(24 * time.Hour)
	s.sessions[sessionToken] = session{
		user:       user,
		expiration: expiration,
	}
	return sessionToken, &expiration, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, sessionToken string) {
	delete(s.sessions, sessionToken)
}

func (s *AuthService) GetSessionUser(ctx context.Context, sessionToken string) (user entity.User, err error) {
	session, exists := s.sessions[sessionToken]
	if !exists {
		return entity.User{}, NoValidSessionError
	}
	if session.isExpired() {
		delete(s.sessions, sessionToken)
		return entity.User{}, NoValidSessionError
	}
	return session.user, nil
}
