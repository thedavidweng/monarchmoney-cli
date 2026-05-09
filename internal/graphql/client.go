package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

// Request represents a GraphQL request.
type Request struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
}

// Client is a Monarch Money GraphQL client.
type Client struct {
	Endpoint string
	Token    string
	HTTP     *http.Client
}

// TokenValue returns the configured auth token.
func (c *Client) TokenValue() string {
	return c.Token
}

// NewClient returns a new Client.
func NewClient(endpoint, token string, timeout time.Duration) *Client {
	return &Client{
		Endpoint: endpoint,
		Token:    token,
		HTTP:     &http.Client{Timeout: timeout},
	}
}

// Do executes a GraphQL request.
func (c *Client) Do(ctx context.Context, reqBody *Request, result interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return errors.New(errors.InternalError, "failed to marshal request", errors.CatInternal, false, err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return errors.New(errors.InternalError, "failed to create request", errors.CatInternal, false, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Platform", "web")
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.Token))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New(errors.NetworkUnreachable, "failed to reach Monarch API", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return errors.New(errors.AuthSessionExpired, "session expired or invalid", errors.CatAuth, true, nil)
	}

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, fmt.Sprintf("API returned status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New(errors.InternalError, "failed to read response body", errors.CatInternal, false, err)
	}

	// Standard GraphQL response wrapper
	var gqlResp struct {
		Data   interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	gqlResp.Data = result

	if err := json.Unmarshal(respData, &gqlResp); err != nil {
		return errors.New(errors.APISchemaChanged, "failed to parse GraphQL response", errors.CatAPI, false, err)
	}

	if len(gqlResp.Errors) > 0 {
		return errors.New(errors.APIError, gqlResp.Errors[0].Message, errors.CatAPI, false, nil)
	}

	return nil
}
