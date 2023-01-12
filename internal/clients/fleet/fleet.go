package fleet

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

type FleetClient struct {
	generatedClient *ClientWithResponses
}

func (c *FleetClient) PostEnrollmentAPIKeys(ctx context.Context, params CreateEnrollmentApiKeysJSONRequestBody) (*EnrollmentApiKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	resp, err := c.generatedClient.CreateEnrollmentApiKeysWithResponse(ctx, params)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if resp.StatusCode() != http.StatusOK {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
	return resp.JSON200.Item, diags
}

func (c *FleetClient) PostAgentPolicy(ctx context.Context, params CreateAgentPolicyJSONRequestBody) (*AgentPolicy, diag.Diagnostics) {
	resp, err := c.generatedClient.CreateAgentPolicyWithResponse(ctx, params)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return resp.JSON200.Item, nil
	default:
		var diags diag.Diagnostics
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
}

func (c *FleetClient) GetAgentPolicy(ctx context.Context, policyID string) (*AgentPolicy, diag.Diagnostics) {
	resp, err := c.generatedClient.AgentPolicyInfoWithResponse(ctx, policyID)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Item, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		var diags diag.Diagnostics
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
}

func (c *FleetClient) DeleteAgentPolicy(ctx context.Context, policyID string) diag.Diagnostics {
	body := DeleteAgentPolicyJSONRequestBody{
		AgentPolicyId: policyID,
	}

	resp, err := c.generatedClient.DeleteAgentPolicyWithResponse(ctx, body)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return nil
	default:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return diags
	}
}

func (c *FleetClient) GetEnrollmentApiKey(ctx context.Context, id string) (*EnrollmentApiKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	resp, err := c.generatedClient.GetEnrollmentApiKeyWithResponse(ctx, id)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Item, diags
	case http.StatusNotFound:
		return nil, nil
	default:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
}

func (c *FleetClient) DeleteEnrollmentApiKey(ctx context.Context, id string) diag.Diagnostics {
	var diags diag.Diagnostics

	resp, err := c.generatedClient.DeleteEnrollmentApiKeyWithResponse(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.StatusCode() != http.StatusOK {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return diags
	}
	return diags
}

func (c *FleetClient) CreatePackagePolicy(ctx context.Context, body CreatePackagePolicyJSONRequestBody) (*PackagePolicy, diag.Diagnostics) {
	resp, err := c.generatedClient.CreatePackagePolicyWithResponse(ctx, body)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Item, nil
	default:
		var diags diag.Diagnostics
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
}

func (c *FleetClient) ReadPackagePolicy(ctx context.Context, policyID string) (*PackagePolicy, diag.Diagnostics) {
	body := BulkGetPackagePoliciesJSONRequestBody{
		Ids: []string{policyID},
	}

	resp, err := c.generatedClient.BulkGetPackagePoliciesWithResponse(ctx, body)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Items[0], nil
	case http.StatusNotFound:
		return nil, nil
	default:
		var diags diag.Diagnostics
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return nil, diags
	}
}

func (c *FleetClient) DeletePackagePolicy(ctx context.Context, id string) diag.Diagnostics {
	body := PostDeletePackagePolicyJSONRequestBody{
		PackagePolicyIds: []string{id},
		Force:            ptrTo(true),
	}

	resp, err := c.generatedClient.PostDeletePackagePolicyWithResponse(ctx, body)
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.StatusCode() != http.StatusOK {
		var diags diag.Diagnostics
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Server responded with HTTP %d", resp.StatusCode()),
			Detail:   string(resp.Body),
		})
		return diags
	}

	return nil
}

func NewFleetClient(config KibanaConfig) (*FleetClient, error) {
	httpClient := &http.Client{
		Transport: &kibanaTransport{config},
	}

	if !strings.HasSuffix(config.URL, "/api/fleet") {
		config.URL += "/api/fleet"
	}

	generatedClient, err := NewClientWithResponses(config.URL, WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &FleetClient{
		generatedClient: generatedClient,
	}, nil
}

type KibanaConfig struct {
	URL      string
	Username string
	Password string
	APIKey   string
	Header   http.Header
}

type kibanaTransport struct {
	KibanaConfig
}

const logReqMsg = `Fleet API Request Details:
---[ REQUEST ]---------------------------------------
%s
-----------------------------------------------------`

const logRespMsg = `Fleet API Response Details:
---[ RESPONSE ]--------------------------------------
%s
-----------------------------------------------------`

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

	if logging.IsDebugOrHigher() {
		if data, err := httputil.DumpRequest(req, true); err == nil {
			tflog.Debug(req.Context(), fmt.Sprintf(logReqMsg, data))
		} else {
			tflog.Debug(req.Context(), fmt.Sprintf("Fleet API request dump error: %#v", err))
		}
	}
	return http.DefaultTransport.RoundTrip(req)
}

func ptrTo[T any](in T) *T {
	return &in
}
