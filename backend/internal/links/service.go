package links

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

const (
	codeLength = 7
	alphabet   = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	maxRetries = 5
)

var ErrInvalidURL = errors.New("invalid url")

type Repository interface {
	Create(ctx context.Context, code, url string) (Link, error)
	GetByCode(ctx context.Context, code string) (Link, error)
	List(ctx context.Context, limit int) ([]Link, error)
}

type Service struct {
	repo    Repository
	baseURL string
}

func NewService(repo Repository, baseURL string) *Service {
	return &Service{repo: repo, baseURL: strings.TrimRight(baseURL, "/")}
}

type CreateInput struct {
	URL string `json:"url"`
}

type CreateResult struct {
	Link
	ShortURL string `json:"short_url"`
}

func (s *Service) Create(ctx context.Context, rawURL string) (CreateResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if err := validateURL(rawURL); err != nil {
		return CreateResult{}, err
	}

	for i := 0; i < maxRetries; i++ {
		code, err := generateCode(codeLength)
		if err != nil {
			return CreateResult{}, fmt.Errorf("generate code: %w", err)
		}
		link, err := s.repo.Create(ctx, code, rawURL)
		if errors.Is(err, ErrCodeExists) {
			continue
		}
		if err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Link: link, ShortURL: s.baseURL + "/r/" + link.Code}, nil
	}
	return CreateResult{}, fmt.Errorf("could not generate unique code after %d attempts", maxRetries)
}

func (s *Service) Get(ctx context.Context, code string) (Link, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *Service) List(ctx context.Context, limit int) ([]Link, error) {
	return s.repo.List(ctx, limit)
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}
	if u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func generateCode(n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}
