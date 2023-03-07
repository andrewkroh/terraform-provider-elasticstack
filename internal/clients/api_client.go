package clients

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type CompositeId struct {
	ClusterId  string
	ResourceId string
}

func CompositeIdFromStr(id string) (*CompositeId, diag.Diagnostics) {
	var diags diag.Diagnostics
	idParts := strings.Split(id, "/")
	if len(idParts) != 2 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Wrong resource ID.",
			Detail:   "Resource ID must have following format: <cluster_uuid>/<resource identifier>",
		})
		return nil, diags
	}
	return &CompositeId{
			ClusterId:  idParts[0],
			ResourceId: idParts[1],
		},
		diags
}

func ResourceIDFromStr(id string) (string, diag.Diagnostics) {
	compID, diags := CompositeIdFromStr(id)
	if diags.HasError() {
		return "", diags
	}
	return compID.ResourceId, nil
}

func (c *CompositeId) String() string {
	return fmt.Sprintf("%s/%s", c.ClusterId, c.ResourceId)
}

type ApiClient struct {
	es      *elasticsearch.Client
	kibana  *kibana.Client
	version string
}

func NewApiClientFunc(version string) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		esClient, diags := newEsApiClient(d, "elasticsearch", version, true)
		if diags.HasError() {
			return nil, diags
		}

		kibanaClient, diags := newKibanaApiClient(d, version)
		if diags.HasError() {
			return nil, diags
		}

		return &ApiClient{es: esClient, kibana: kibanaClient, version: version}, nil
	}
}

func NewAcceptanceTestingClient() (*ApiClient, error) {
	config := elasticsearch.Config{}
	config.Header = http.Header{"User-Agent": []string{"elasticstack-terraform-provider/tf-acceptance-testing"}}

	if es := os.Getenv("ELASTICSEARCH_ENDPOINTS"); es != "" {
		endpoints := make([]string, 0)
		for _, e := range strings.Split(es, ",") {
			endpoints = append(endpoints, strings.TrimSpace(e))
		}
		config.Addresses = endpoints
	}

	if username := os.Getenv("ELASTICSEARCH_USERNAME"); username != "" {
		config.Username = username
		config.Password = os.Getenv("ELASTICSEARCH_PASSWORD")
	} else {
		config.APIKey = os.Getenv("ELASTICSEARCH_API_KEY")
	}

	es, err := elasticsearch.NewClient(config)
	if err != nil {
		return nil, err
	}

	// TODO:
	_, insecure := os.LookupEnv("KIBANA_INSECURE")
	kibana := kibana.NewClient(kibana.Config{
		URL:      os.Getenv("KIBANA_ENDPOINTS"),
		Username: os.Getenv("KIBANA_USERNAME"),
		Password: os.Getenv("KIBANA_PASSWORD"),
		APIKey:   os.Getenv("KIBANA_APIKEY"),
		Insecure: insecure,
	})

	return &ApiClient{es, kibana, "acceptance-testing"}, nil
}

const (
	esConnectionKey = "elasticsearch_connection"
)

func NewApiClient(d *schema.ResourceData, meta interface{}) (*ApiClient, diag.Diagnostics) {
	defaultClient := *meta.(*ApiClient)

	if _, ok := d.GetOk(esConnectionKey); ok {
		es, diag := newEsApiClient(d, esConnectionKey, defaultClient.version, false)
		if diag.HasError() {
			return nil, diag
		}
		defaultClient.es = es
	}

	return &defaultClient, nil
}

func ensureTLSClientConfig(config *elasticsearch.Config) *tls.Config {
	if config.Transport == nil {
		config.Transport = http.DefaultTransport.(*http.Transport)
	}
	if config.Transport.(*http.Transport).TLSClientConfig == nil {
		config.Transport.(*http.Transport).TLSClientConfig = &tls.Config{}
	}
	return config.Transport.(*http.Transport).TLSClientConfig
}

func (a *ApiClient) GetESClient() *elasticsearch.Client {
	return a.es
}

func (a *ApiClient) GetKibanaClient() *kibana.Client {
	return a.kibana
}

func (a *ApiClient) ID(ctx context.Context, resourceId string) (*CompositeId, diag.Diagnostics) {
	var diags diag.Diagnostics
	clusterId, diags := a.ClusterID(ctx)
	if diags.HasError() {
		return nil, diags
	}
	return &CompositeId{*clusterId, resourceId}, diags
}

func (a *ApiClient) serverInfo(ctx context.Context) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	res, err := a.es.Info(a.es.Info.WithContext(ctx))
	if err != nil {
		return nil, diag.FromErr(err)
	}
	defer res.Body.Close()
	if diags := utils.CheckError(res, "Unable to connect to the Elasticsearch cluster"); diags.HasError() {
		return nil, diags
	}

	info := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, diag.FromErr(err)
	}

	return info, diags
}

func (a *ApiClient) ServerVersion(ctx context.Context) (*version.Version, diag.Diagnostics) {
	info, diags := a.serverInfo(ctx)
	if diags.HasError() {
		return nil, diags
	}

	rawVersion := info["version"].(map[string]interface{})["number"].(string)
	serverVersion, err := version.NewVersion(rawVersion)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return serverVersion, nil
}

func (a *ApiClient) ClusterID(ctx context.Context) (*string, diag.Diagnostics) {
	info, diags := a.serverInfo(ctx)
	if diags.HasError() {
		return nil, diags
	}

	if uuid := info["cluster_uuid"].(string); uuid != "" && uuid != "_na_" {
		tflog.Trace(ctx, fmt.Sprintf("cluster UUID: %s", uuid))
		return &uuid, diags
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "Unable to get cluster UUID",
		Detail: `Unable to get cluster UUID.
		There might be a problem with permissions or cluster is still starting up and UUID has not been populated yet.`,
	})
	return nil, diags
}

func newEsApiClient(d *schema.ResourceData, key string, version string, useEnvAsDefault bool) (*elasticsearch.Client, diag.Diagnostics) {
	var diags diag.Diagnostics
	config := elasticsearch.Config{}
	config.Header = http.Header{"User-Agent": []string{fmt.Sprintf("elasticstack-terraform-provider/%s", version)}}

	if esConn, ok := d.GetOk(key); ok {
		// if defined, then we only have a single entry
		if es := esConn.([]interface{})[0]; es != nil {
			esConfig := es.(map[string]interface{})

			if username, ok := esConfig["username"]; ok {
				config.Username = username.(string)
			}
			if password, ok := esConfig["password"]; ok {
				config.Password = password.(string)
			}
			if apikey, ok := esConfig["api_key"]; ok {
				config.APIKey = apikey.(string)
			}

			if useEnvAsDefault {
				if endpoints := os.Getenv("ELASTICSEARCH_ENDPOINTS"); endpoints != "" {
					var addrs []string
					for _, e := range strings.Split(endpoints, ",") {
						addrs = append(addrs, strings.TrimSpace(e))
					}
					config.Addresses = addrs
				}
			}

			if endpoints, ok := esConfig["endpoints"]; ok && len(endpoints.([]interface{})) > 0 {
				var addrs []string
				for _, e := range endpoints.([]interface{}) {
					addrs = append(addrs, e.(string))
				}
				config.Addresses = addrs
			}

			if insecure, ok := esConfig["insecure"]; ok && insecure.(bool) {
				tlsClientConfig := ensureTLSClientConfig(&config)
				tlsClientConfig.InsecureSkipVerify = true
			}

			if caFile, ok := esConfig["ca_file"]; ok && caFile.(string) != "" {
				caCert, err := os.ReadFile(caFile.(string))
				if err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Unable to read CA File",
						Detail:   err.Error(),
					})
					return nil, diags
				}
				config.CACert = caCert
			}
			if caData, ok := esConfig["ca_data"]; ok && caData.(string) != "" {
				config.CACert = []byte(caData.(string))
			}

			if certFile, ok := esConfig["cert_file"]; ok && certFile.(string) != "" {
				if keyFile, ok := esConfig["key_file"]; ok && keyFile.(string) != "" {
					cert, err := tls.LoadX509KeyPair(certFile.(string), keyFile.(string))
					if err != nil {
						diags = append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Unable to read certificate or key file",
							Detail:   err.Error(),
						})
						return nil, diags
					}
					tlsClientConfig := ensureTLSClientConfig(&config)
					tlsClientConfig.Certificates = []tls.Certificate{cert}
				} else {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Unable to read key file",
						Detail:   "Path to key file has not been configured or is empty",
					})
					return nil, diags
				}
			}
			if certData, ok := esConfig["cert_data"]; ok && certData.(string) != "" {
				if keyData, ok := esConfig["key_data"]; ok && keyData.(string) != "" {
					cert, err := tls.X509KeyPair([]byte(certData.(string)), []byte(keyData.(string)))
					if err != nil {
						diags = append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Unable to parse certificate or key",
							Detail:   err.Error(),
						})
						return nil, diags
					}
					tlsClientConfig := ensureTLSClientConfig(&config)
					tlsClientConfig.Certificates = []tls.Certificate{cert}
				} else {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Unable to parse key",
						Detail:   "Key data has not been configured or is empty",
					})
					return nil, diags
				}
			}
		}
	}

	es, err := elasticsearch.NewClient(config)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Elasticsearch client",
			Detail:   err.Error(),
		})
		return nil, diags
	}
	if logging.IsDebugOrHigher() {
		es.Transport = newDebugTransport("elasticsearch", es.Transport)
	}

	return es, diags
}

func newKibanaApiClient(d *schema.ResourceData, version string) (*kibana.Client, diag.Diagnostics) {
	// TODO: Is version param needed?
	const providerName = "kibana"

	var diags diag.Diagnostics
	config := kibana.Config{
		Header: http.Header{"User-Agent": []string{fmt.Sprintf("elasticstack-terraform-provider/%s", providerName)}},
	}

	if connSchema, ok := d.GetOk(providerName); ok {
		// if defined, then we only have a single entry
		if es := connSchema.([]interface{})[0]; es != nil {
			kibanaConfig := es.(map[string]interface{})
			tflog.Error(context.Background(), spew.Sdump(kibanaConfig))

			if urls, ok := kibanaConfig["endpoints"]; ok {
				// TODO: Make client support multiple endpoints.
				config.URL = urls.(string)
			}
			if username, ok := kibanaConfig["username"]; ok {
				config.Username = username.(string)
			}
			if password, ok := kibanaConfig["password"]; ok {
				config.Password = password.(string)
			}
			if apikey, ok := kibanaConfig["api_key"]; ok {
				config.APIKey = apikey.(string)
			}
			if v, ok := kibanaConfig["insecure"]; ok {
				config.Insecure = v.(bool)
			}

			// TODO: Pass through all the TLS options.
		}
	}

	return kibana.NewClient(config), diags
}
