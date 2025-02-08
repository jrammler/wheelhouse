package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
)

type mockStorage struct {
	user entity.User
	err  error
}

func (m *mockStorage) GetCommands(ctx context.Context) ([]entity.Command, error) {
	return nil, errors.New("not supported")
}

func (m *mockStorage) GetCommandById(ctx context.Context, id string) (*entity.Command, error) {
	return nil, errors.New("not supported")
}

func (m *mockStorage) GetUser(ctx context.Context, username string) (entity.User, error) {
	if m.err != nil {
		return entity.User{}, m.err
	}
	return m.user, nil
}

func (m *mockStorage) LoadConfig() error {
	return nil
}

func TestLoginUser(t *testing.T) {
	testCases := []struct {
		name          string
		username      string
		password      string
		storage       mockStorage
		expectedError error
	}{
		{
			name:     "Successful login",
			username: "testuser",
			password: "password",
			storage: mockStorage{
				user: entity.User{
					Username:     "testuser",
					PasswordHash: "$2a$04$dKD7Ty3vN6sYhWyRxDKepOOsjbJ2HtU/Q0Dw7wt.5Q2cqCXJEi/Wa", // "password"
				},
				err: nil,
			},
			expectedError: nil,
		},
		{
			name:     "Invalid credentials",
			username: "testuser",
			password: "wrongpassword",
			storage: mockStorage{
				user: entity.User{
					Username:     "testuser",
					PasswordHash: "$2a$04$dKD7Ty3vN6sYhWyRxDKepOOsjbJ2HtU/Q0Dw7wt.5Q2cqCXJEi/Wa", // "password"
				},
				err: nil,
			},
			expectedError: CredentialError,
		},
		{
			name:          "User not found",
			username:      "testuser",
			password:      "password",
			storage:       mockStorage{err: errors.New("user not found")},
			expectedError: CredentialError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authService := NewAuthService(&tc.storage)
			_, _, err := authService.LoginUser(context.Background(), tc.username, tc.password)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("Expected %q, got %q", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error %q", err)
				}
			}
		})
	}
}

func TestLogoutUser(t *testing.T) {
	authService := NewAuthService(&mockStorage{})
	token, err := generateSessionToken()
	if err != nil {
		t.Errorf("Unexpected error %q", err)
	}
	authService.sessions[token] = session{
		user:       entity.User{Username: "test"},
		expiration: time.Now().Add(time.Hour),
	}
	authService.LogoutUser(context.Background(), token)
	_, exists := authService.sessions[token]
	if exists {
		t.Errorf("Found token but expected it to be deleted")
	}
}

func TestGetSessionUser(t *testing.T) {
	testCases := []struct {
		name          string
		sessionToken  string
		setup         func(as *AuthService, token string)
		expectedUser  entity.User
		expectedError error
	}{
		{
			name:         "Valid session",
			sessionToken: "valid_token",
			setup: func(as *AuthService, token string) {
				as.sessions[token] = session{
					user:       entity.User{Username: "test"},
					expiration: time.Now().Add(time.Hour),
				}
			},
			expectedUser:  entity.User{Username: "test"},
			expectedError: nil,
		},
		{
			name:         "No valid session",
			sessionToken: "invalid_token",
			setup: func(as *AuthService, token string) {
			},
			expectedUser:  entity.User{},
			expectedError: NoValidSessionError,
		},
		{
			name:         "Expired session",
			sessionToken: "expired_token",
			setup: func(as *AuthService, token string) {
				as.sessions[token] = session{
					user:       entity.User{Username: "test"},
					expiration: time.Now().Add(-time.Hour),
				}
			},
			expectedUser:  entity.User{},
			expectedError: NoValidSessionError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authService := NewAuthService(&mockStorage{})
			tc.setup(authService, tc.sessionToken)
			user, err := authService.GetSessionUser(context.Background(), tc.sessionToken)

			if user.Username != tc.expectedUser.Username {
				t.Errorf("Expected username %q but got %q", tc.expectedUser.Username, user.Username)
			}
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error %q, expected %q", err, tc.expectedError)
			}
		})
	}
}
