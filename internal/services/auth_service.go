package services

import (
	"errors"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/school-system/backend/internal/config"
	"github.com/school-system/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user not active")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenRevoked       = errors.New("token revoked")
)

type AuthService struct {
	db     *gorm.DB
	cfg    *config.Config
	params *argon2id.Params
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Claims struct {
	UserID   uuid.UUID  `json:"user_id"`
	SchoolID *uuid.UUID `json:"school_id"`
	Role     string     `json:"role"`
	Email    string     `json:"email"`
	jwt.RegisteredClaims
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	params := &argon2id.Params{
		Memory:      cfg.Argon2.Memory,
		Iterations:  cfg.Argon2.Iterations,
		Parallelism: cfg.Argon2.Parallelism,
		SaltLength:  cfg.Argon2.SaltLength,
		KeyLength:   cfg.Argon2.KeyLength,
	}

	return &AuthService{
		db:     db,
		cfg:    cfg,
		params: params,
	}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, s.params)
}

func (s *AuthService) VerifyPassword(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func (s *AuthService) Login(email, password string) (*TokenPair, *models.User, error) {
	var user models.User
	if err := s.db.Preload("School").Where("LOWER(email) = LOWER(?)", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	match, err := s.VerifyPassword(user.PasswordHash, password)
	if err != nil || !match {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.GenerateTokenPair(&user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, &user, nil
}

func (s *AuthService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	// Access token
	accessClaims := &Claims{
		UserID:   user.ID,
		SchoolID: user.SchoolID,
		Role:     user.Role,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWT.AccessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshClaims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWT.RefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, err
	}

	// Store refresh token
	rt := &models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshExpiry),
		Revoked:   false,
	}
	if err := s.db.Create(rt).Error; err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.cfg.JWT.AccessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) RefreshTokens(refreshToken string) (*TokenPair, error) {
	// Verify token
	claims, err := s.VerifyToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Check if token is revoked
	var rt models.RefreshToken
	if err := s.db.Where("token = ?", refreshToken).First(&rt).Error; err != nil {
		return nil, ErrInvalidToken
	}

	if rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return nil, ErrTokenRevoked
	}

	// Get user
	var user models.User
	if err := s.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	// Revoke old token
	s.db.Model(&rt).Update("revoked", true)

	// Generate new pair
	return s.GenerateTokenPair(&user)
}

func (s *AuthService) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) RevokeToken(refreshToken string) error {
	return s.db.Model(&models.RefreshToken{}).
		Where("token = ?", refreshToken).
		Update("revoked", true).Error
}

func (s *AuthService) CreateUser(user *models.User, password string) error {
	hash, err := s.HashPassword(password)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	return s.db.Create(user).Error
}
