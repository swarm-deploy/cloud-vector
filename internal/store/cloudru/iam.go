package cloudru

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	iamAuthV1 "github.com/cloudru-tech/iam-sdk/api/auth/v1"
	"github.com/swarm-deploy/cloud-vector/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type iam struct {
	iamClient iamAuthV1.AuthServiceClient

	mu                   sync.Mutex
	accessToken          string
	accessTokenExpiresAt time.Time

	accessKey    string
	accessSecret string
}

func newIAM(ctx context.Context, cfg config.Cloudru) (*iam, error) {
	address := cfg.IAM.Address
	if address == "" {
		endpoints, err := getEndpoints(ctx, cfg.DiscoveryURL)
		if err != nil {
			return nil, fmt.Errorf("get cloud.ru endpoints: %w", err)
		}

		iamEndpoint := endpoints.Get("iam")
		if iamEndpoint == nil {
			return nil, errors.New("iam endpoint not found")
		}

		address = iamEndpoint.Address
	}

	iamConn, err := grpc.NewClient(address, grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13}),
	))
	if err != nil {
		return nil, fmt.Errorf("create iam grpc client: %w", err)
	}

	return &iam{
		iamClient:    iamAuthV1.NewAuthServiceClient(iamConn),
		accessKey:    cfg.IAM.ClientID,
		accessSecret: cfg.IAM.ClientSecret,
	}, nil
}

func (i *iam) Token(ctx context.Context) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.accessToken != "" && i.accessTokenExpiresAt.After(time.Now()) {
		return i.accessToken, nil
	}

	slog.InfoContext(ctx, "[iam] request new token")

	resp, err := i.iamClient.GetToken(ctx, &iamAuthV1.GetTokenRequest{KeyId: i.accessKey, Secret: i.accessSecret})
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}

	slog.InfoContext(ctx, "[iam] new token fetched")

	i.accessToken = resp.AccessToken
	i.accessTokenExpiresAt = time.Now().Add(time.Second * time.Duration(resp.ExpiresIn))
	return i.accessToken, nil
}
