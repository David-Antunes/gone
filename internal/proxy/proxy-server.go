package proxy

import (
	"net/http"
)

type ProxyServer struct {
	client   *http.Client
	endpoint string
}

func NewProxyServer(client *http.Client, endpoint string) *ProxyServer {
	return &ProxyServer{
		client:   client,
		endpoint: endpoint,
	}
}

func (p *ProxyServer) Refresh() error {
	_, err := p.client.Get("http://unix" + "/refresh")
	if err != nil {
		return err
	}
	return nil
}
