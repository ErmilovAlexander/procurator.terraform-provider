package client

func (c *grpcClient) AccessToken() string {
	return c.cfg.Token
}
