package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultURL    = "https://cockroachlabs.cloud/api/v1"
	jsonMediaType = "application/json"
)

type ClientOption func(c *Client) error

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("error parsing URL: %v", err)
		}

		c.baseURL = parsedURL
		return nil
	}
}

func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) error {
		if client == nil {
			client = http.DefaultClient
		}

		c.client = client
		return nil
	}
}

func WithAccessToken(accessToken string) ClientOption {
	return func(c *Client) error {
		if accessToken == "" {
			return fmt.Errorf("access token must not be empty")
		}
		if c.client.Transport == nil {
			c.client.Transport = http.DefaultTransport
		}

		c.client.Transport = &accessTokenTransport{
			rt:          c.client.Transport,
			accessToken: accessToken,
		}
		return nil
	}
}

type Client struct {
	client  *http.Client
	baseURL *url.URL

	Cluster *ClusterClient
}

func NewClient(opts ...ClientOption) (*Client, error) {
	baseURL, err := url.Parse(defaultURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	client := Client{
		client:  http.DefaultClient,
		baseURL: baseURL,
	}
	for _, opt := range opts {
		if err := opt(&client); err != nil {
			return nil, fmt.Errorf("error setting option: %v", err)
		}
	}

	client.Cluster = &ClusterClient{
		client: &client,
	}

	return &client, nil
}

func (c *Client) do(ctx context.Context, req *http.Request, val interface{}) error {
	reqWithCtx := req.WithContext(ctx)
	res, err := c.client.Do(reqWithCtx)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer res.Body.Close()

	return c.handleResponse(ctx, res, val)
}

func (c *Client) handleResponse(ctx context.Context, res *http.Response, val interface{}) error {
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	if res.StatusCode >= 400 {
		var errResponse errorResponse
		if err := json.Unmarshal(bytes, &errResponse); err != nil {
			return fmt.Errorf("error umarshalling response error: %v", err)
		}
		return &Error{
			ErrorCode: errResponse.Code,
			HTTPCode:  res.StatusCode,
			Message:   errResponse.Message,
		}
	}

	if val == nil {
		return nil
	}
	if err := json.Unmarshal(bytes, &val); err != nil {
		return fmt.Errorf("error umarshalling response error: %v", err)
	}
	return nil
}

func (c *Client) newRequest(method, path string, queryParams map[string]string, body interface{}) (*http.Request, error) {
	url, err := buildURL(*c.baseURL, path, queryParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}
	setHeaders := func(req *http.Request) {
		req.Header.Set("Content-Type", jsonMediaType)
		req.Header.Set("Accept", jsonMediaType)
	}

	if method == http.MethodGet {
		req, err := http.NewRequest(method, url.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}
		setHeaders(req)
		return req, nil
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	setHeaders(req)
	return req, nil
}

func buildURL(baseUrl url.URL, path string, queryParams map[string]string) (*url.URL, error) {
	if !strings.HasPrefix(path, "/") {
		baseUrl.Path += "/"
	}
	baseUrl.Path += path

	if queryParams != nil {
		query := baseUrl.Query()
		for k, v := range queryParams {
			query.Set(k, v)
		}
		baseUrl.RawQuery = query.Encode()
	}

	newUrl, err := baseUrl.Parse(baseUrl.String())
	if err != nil {
		return nil, fmt.Errorf("error appending path to URL: %v", err)
	}
	return newUrl, nil
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Error struct {
	ErrorCode int
	HTTPCode  int
	Message   string
}

func (e *Error) Error() string {
	return e.Message
}

type accessTokenTransport struct {
	rt          http.RoundTripper
	accessToken string
}

func (t *accessTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.accessToken)
	return t.rt.RoundTrip(req)
}
