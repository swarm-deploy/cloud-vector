package cloudru

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/swarm-deploy/cloud-vector/internal/config"
)

type Store struct {
	cfg config.Cloudru

	iam        *iam
	httpClient *http.Client
}

func NewStore(ctx context.Context, cfg config.Cloudru) (*Store, error) {
	iamClient, err := newIAM(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create iam client: %w", err)
	}

	return &Store{
		cfg:        cfg,
		iam:        iamClient,
		httpClient: http.DefaultClient,
	}, nil
}

func (s *Store) Push(ctx context.Context, logs []interface{}) error {
	// Оборачиваем в объект с ключом "logs"
	wrapped := map[string]interface{}{
		"logs": logs,
	}

	// Сериализуем обратно в JSON
	wrappedJSON, err := json.Marshal(wrapped)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Logging.Endpoint, bytes.NewBuffer(wrappedJSON))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}

	token, err := s.iam.Token(ctx)
	if err != nil {
		return fmt.Errorf("get iam token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req) //nolint:gosec // request URL comes from trusted configuration
	if err != nil {
		return fmt.Errorf("send http request to logging: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.WarnContext(ctx, "[proxy][cloudru] failed to read response body", slog.Any("err", err))
	} else {
		slog.InfoContext(ctx, "[proxy][cloudru] server returns response", slog.Any("resp", string(respBody)))
	}

	return nil
}
