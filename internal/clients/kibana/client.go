package kibana

import (
	"crypto/tls"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

type Config struct {
	URL      string
	Username string
	Password string
	APIKey   string
	Insecure bool
	Header   http.Header

	// TODO: Add TLS options
}

type Client struct {
	URL  string
	HTTP *http.Client
}

func NewClient(c Config) *Client {
	var roundTripper http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.Insecure,
		},
	}

	if logging.IsDebugOrHigher() {
		roundTripper = newDebugTransport("Kibana", roundTripper)
	}

	httpClient := &http.Client{
		Transport: &kibanaTransport{
			Config: c,
			next:   roundTripper,
		},
	}

	return &Client{
		URL:  c.URL,
		HTTP: httpClient,
	}
}

type kibanaTransport struct {
	Config
	next http.RoundTripper
}

func (t *kibanaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case "GET", "HEAD":
	default:
		// https://www.elastic.co/guide/en/kibana/current/api.html#api-request-headers
		req.Header.Add("kbn-xsrf", "true")
	}

	if t.Username != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}

	if t.APIKey != "" {
		req.Header.Add("Authorization", "Bearer "+t.APIKey)
	}

	return t.next.RoundTrip(req)
}
