package apiclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func (c *Client) ListPATs(ctx context.Context, accessToken string) ([]domain.PAT, error) {
	var result struct {
		Data []domain.PAT `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/pats", accessToken, nil, &result)
	return result.Data, err
}

func (c *Client) CreatePAT(ctx context.Context, accessToken, name string) (domain.CreatedPAT, error) {
	var result domain.CreatedPAT
	err := c.do(ctx, http.MethodPost, "/v1/pats", accessToken, map[string]string{"name": name}, &result)
	return result, err
}

func (c *Client) RevokePAT(ctx context.Context, accessToken, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/pats/"+url.PathEscape(id), accessToken, nil, nil)
}
