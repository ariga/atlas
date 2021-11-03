package client

type Client struct{}

func New(dsn string) (*Client, error) {
	return &Client{}, nil
}
