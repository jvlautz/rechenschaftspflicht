package authentication

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"time"

	"github.com/erkannt/rechenschaftspflicht/services/config"
	"github.com/golang-jwt/jwt/v4"
)

// Auth defines the public contract for the service.
type Auth interface {
	GenerateToken(email string) (string, error)
	ValidateToken(tokenStr string) (string, error)
	SendMagicLink(toEmail, token string) error
	IsLoggedIn(r *http.Request) bool
	GetLoggedInUserEmail(r *http.Request) (string, error)
	LoggedIn(token string) http.Cookie
	LoggedOut() http.Cookie
}

// magicLinksSvc is the concrete implementation holding internal state.
type magicLinksSvc struct {
	jwtSecret []byte
	smtpAuth  smtp.Auth
	smtpFrom  string
	smtpAddr  string
	appOrigin string
	isHTTPS   bool
}

func createSmtpAuth(logger *slog.Logger, cfg config.Config) smtp.Auth {
	if cfg.SMTPUser == "" || cfg.SMTPUser == `""` {
		logger.Warn("using SMTP without authentication as SMTPUSER is an empty string")
		return nil
	}

	return smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
}

func New(logger *slog.Logger, cfg config.Config) Auth {
	return &magicLinksSvc{
		jwtSecret: []byte(cfg.JWTSecret),
		smtpAuth:  createSmtpAuth(logger, cfg),
		smtpFrom:  cfg.SMTPFrom,
		smtpAddr:  fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort),
		appOrigin: cfg.AppOrigin,
		isHTTPS:   isHTTPS(cfg.AppOrigin),
	}
}

// isHTTPS determines if the origin uses HTTPS protocol
func isHTTPS(origin string) bool {
	return len(origin) > 5 && origin[:5] == "https"
}

func (s *magicLinksSvc) GenerateToken(email string) (string, error) {
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

// ValidateToken parses and validates the JWT, returning the embedded email if valid.
func (s *magicLinksSvc) ValidateToken(input string) (string, error) {
	token, err := jwt.Parse(input, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if email, ok := claims["email"].(string); ok {
			return email, nil
		}
	}
	return "", fmt.Errorf("email claim missing")
}

// SendMagicLink sends an email containing a login link with the supplied token.
func (s *magicLinksSvc) SendMagicLink(toEmail, token string) error {
	if s.smtpFrom == "" {
		return fmt.Errorf("SMTP configuration incomplete: missing from address")
	}

	link := fmt.Sprintf("%s/login?token=%s", s.appOrigin, token)
	msg := fmt.Sprintf(
		"From: %s\r\nSubject: Your Magic Login Link\r\n\r\nClick the following link to log in:\n\n%s",
		s.smtpFrom,
		link,
	)

	return smtp.SendMail(s.smtpAddr, s.smtpAuth, s.smtpFrom, []string{toEmail}, []byte(msg))
}

func (s *magicLinksSvc) IsLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("auth")
	if err != nil || cookie.Value == "" {
		return false
	}
	if email, err := s.ValidateToken(cookie.Value); err != nil || email == "" {
		return false
	}
	return true
}

func (s *magicLinksSvc) GetLoggedInUserEmail(r *http.Request) (string, error) {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return "", err
	}
	if cookie.Value == "" {
		return "", http.ErrNoCookie
	}
	email, err := s.ValidateToken(cookie.Value)
	if err != nil {
		return "", err
	}
	return email, nil
}

func (s *magicLinksSvc) LoggedIn(token string) http.Cookie {
	return http.Cookie{
		Name:     "auth",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   s.isHTTPS,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *magicLinksSvc) LoggedOut() http.Cookie {
	return http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire immediately
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.isHTTPS,
		SameSite: http.SameSiteLaxMode,
	}
}
