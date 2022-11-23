package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gempages/go-helper/tracing"
	"github.com/gempages/go-shopify-graphql/utils"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
	"golang.org/x/net/context/ctxhttp"
)

// Client is a GraphQL client.
type Client struct {
	url        string // GraphQL server URL.
	httpClient *http.Client
}

// NewClient creates a GraphQL client targeting the specified GraphQL server URL.
// If httpClient is nil, then http.DefaultClient is used.
func NewClient(url string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		url:        url,
		httpClient: httpClient,
	}
}

// QueryString executes a single GraphQL query request,
// using the given raw query `q` and populating the response into the `v`.
// `q` should be a correct GraphQL request string that corresponds to the GraphQL schema.
func (c *Client) QueryString(ctx context.Context, q string, variables map[string]interface{}, v interface{}) error {
	return c.do(ctx, q, variables, v)
}

// Query executes a single GraphQL query request,
// with a query derived from q, populating the response into it.
// q should be a pointer to struct that corresponds to the GraphQL schema.
func (c *Client) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	query := constructQuery(q, variables)

	return c.do(ctx, query, variables, q)
}

// Mutate executes a single GraphQL mutation request,
// with a mutation derived from m, populating the response into it.
// m should be a pointer to struct that corresponds to the GraphQL schema.
func (c *Client) Mutate(ctx context.Context, m interface{}, variables map[string]interface{}) error {
	query := constructMutation(m, variables)
	// return nil
	return c.do(ctx, query, variables, m)
}

// do executes a single GraphQL operation.
func (c *Client) do(ctx context.Context, query string, variables map[string]interface{}, v interface{}) error {
	var err error
	in := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}

	// sentry tracing
	span := sentry.StartSpan(ctx, "shopify_graphql.send")
	span.Description = utils.GetDescriptionFromQuery(query)
	span.SetTag("query", query)
	span.SetTag("variables", cast.ToString(variables))
	span.SetTag("url", c.url)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	// end sentry tracing

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(in)
	if err != nil {
		return err
	}
	resp, err := ctxhttp.Post(ctx, c.httpClient, c.url, "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-200 OK status code: %v body: %q", resp.Status, body)
	}
	var out struct {
		Data   *json.RawMessage
		Errors errors
		//Extensions interface{} // Unused.
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		// TODO: Consider including response body in returned error, if deemed helpful.
		return err
	}
	// xx := make(map[string]interface{})
	if out.Data != nil {
		err := json.Unmarshal(*out.Data, v)
		if err != nil {
			// TODO: Consider including response body in returned error, if deemed helpful.
			return err
		}
	}
	if len(out.Errors) > 0 {
		return out.Errors
	}
	return nil
}

// errors represents the "errors" array in a response from a GraphQL server.
// If returned via error interface, the slice is expected to contain at least 1 element.
//
// Specification: https://facebook.github.io/graphql/#sec-Errors.
type errors []struct {
	Message   string
	Locations []struct {
		Line   int
		Column int
	}
}

// Error implements error interface.
func (e errors) Error() string {
	return e[0].Message
}

type operationType uint8

const (
	queryOperation operationType = iota
	mutationOperation
	//subscriptionOperation // Unused.
)
