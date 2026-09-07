package apiclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// DescribeOAuthClient loads a registered MCP client so the consent screen can
// name it. The name must come from here rather than from the query string the
// screen is rendering, or the page would show whatever an attacker put in the
// URL as though the server had vouched for it.
func (c *Client) DescribeOAuthClient(ctx context.Context, accessToken, clientID string) (domain.OAuthClientInfo, error) {
	var info domain.OAuthClientInfo
	err := c.do(ctx, http.MethodGet, "/v1/oauth/clients/"+url.PathEscape(clientID), accessToken, nil, &info)
	return info, err
}

// ApproveOAuthGrant records the user's consent and returns where to send the
// browser next.
func (c *Client) ApproveOAuthGrant(ctx context.Context, accessToken string, input domain.ApproveOAuthGrant) (domain.OAuthGrantApproval, error) {
	var out domain.OAuthGrantApproval
	err := c.do(ctx, http.MethodPost, "/v1/oauth/authorize", accessToken, input, &out)
	return out, err
}
