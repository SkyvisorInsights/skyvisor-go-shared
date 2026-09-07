package apiclient

import (
	"context"
	"net/http"
	"strconv"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// AgentInbox drains pending watch notifications for the account, oldest first.
// A limit of zero lets the server choose.
func (c *Client) AgentInbox(ctx context.Context, accessToken string, limit int) (domain.AgentInboxPage, error) {
	path := "/v1/agent/inbox"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var page domain.AgentInboxPage
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &page)
	return page, err
}

// AckAgentInbox acknowledges events so a later drain does not return them.
func (c *Client) AckAgentInbox(ctx context.Context, accessToken string, input domain.AckAgentInbox) (domain.AgentInboxAck, error) {
	var ack domain.AgentInboxAck
	err := c.do(ctx, http.MethodPost, "/v1/agent/inbox/ack", accessToken, input, &ack)
	return ack, err
}
