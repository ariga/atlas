// The code in this file is a slightly modified version of https://github.com/Khan/genqlient/blob/main/graphql/client.go.
// It is copied here to avoid a dependency on genqlient.

package cloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

// GQLClient is the interface that the generated code calls into to actually make
// requests.
type gqlClient interface {
	// MakeRequest must make a request to the client's GraphQL API.
	//
	// ctx is the context that should be used to make this request.  If context
	// is disabled in the genqlient settings, this will be set to
	// context.Background().
	//
	// req contains the data to be sent to the GraphQL server.  Typically GraphQL
	// APIs will expect it to simply be marshalled as JSON, but MakeRequest may
	// customize this.
	//
	// resp is the Response object into which the server's response will be
	// unmarshalled. Typically GraphQL APIs will return JSON which can be
	// unmarshalled directly into resp, but MakeRequest can customize it.
	// If the response contains an error, this must also be returned by
	// MakeRequest.  The field resp.Data will be prepopulated with a pointer
	// to an empty struct of the correct generated type (e.g. MyQueryResponse).
	MakeRequest(
		ctx context.Context,
		req *Request,
		resp *Response,
	) error
}

type client struct {
	httpClient Doer
	endpoint   string
}

// newGQLClient returns a [Client] which makes requests to the given endpoint,
// suitable for most users.
//
// The client makes POST requests to the given GraphQL endpoint using standard
// GraphQL HTTP-over-JSON transport.  It will use the given [http.Client], or
// [http.DefaultClient] if a nil client is passed.
//
// The typical method of adding authentication headers is to wrap the client's
// [http.Transport] to add those headers.  See [example/main.go] for an
// example.
//
// [example/main.go]: https://github.com/Khan/genqlient/blob/main/example/main.go#L12-L20
func newGQLClient(endpoint string, httpClient Doer) gqlClient {
	if httpClient == nil || httpClient == (*http.Client)(nil) {
		httpClient = http.DefaultClient
	}
	return &client{httpClient, endpoint}
}

// Doer encapsulates the methods from [*http.Client] needed by [Client].
// The methods should have behavior to match that of [*http.Client]
// (or mocks for the same).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Request contains all the values required to build queries executed by
// the [Client].
//
// Typically, GraphQL APIs will accept a JSON payload of the form
//
//	{"query": "query myQuery { ... }", "variables": {...}}`
//
// and Request marshals to this format.  However, MakeRequest may
// marshal the data in some other way desired by the backend.
type Request struct {
	// The literal string representing the GraphQL query, e.g.
	// `query myQuery { myField }`.
	Query string `json:"query"`
	// A JSON-marshalable value containing the variables to be sent
	// along with the query, or nil if there are none.
	Variables interface{} `json:"variables,omitempty"`
	// The GraphQL operation name. The server typically doesn't
	// require this unless there are multiple queries in the
	// document, but genqlient sets it unconditionally anyway.
	OpName string `json:"operationName"`
}

// Response that contains data returned by the GraphQL API.
//
// Typically, GraphQL APIs will return a JSON payload of the form
//
//	{"data": {...}, "errors": {...}}
//
// It may additionally contain a key named "extensions", that
// might hold GraphQL protocol extensions. Extensions and Errors
// are optional, depending on the values returned by the server.
type Response struct {
	Data       interface{}            `json:"data"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	Errors     gqlerror.List          `json:"errors,omitempty"`
}

func (c *client) MakeRequest(ctx context.Context, req *Request, resp *Response) error {
	var httpReq *http.Request
	var err error

	httpReq, err = c.createPostRequest(req)

	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if ctx != nil {
		httpReq = httpReq.WithContext(ctx)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		var respBody []byte
		respBody, err = io.ReadAll(httpResp.Body)
		if err != nil {
			respBody = []byte(fmt.Sprintf("<unreadable: %v>", err))
		}
		return fmt.Errorf("returned error %v: %s", httpResp.Status, respBody)
	}

	err = json.NewDecoder(httpResp.Body).Decode(resp)
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors
	}
	return nil
}

func (c *client) createPostRequest(req *Request) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		c.endpoint,
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return httpReq, nil
}
