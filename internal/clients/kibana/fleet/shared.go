package fleet

import (
	"fmt"
	"strings"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet/fleetapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// fleetClient returns a fleet API specific client built from the kibana.Client.
// It reuses the underlying http.Client from the kibana.Client.
func fleetClient(kibanaClient *kibana.Client) (*fleetapi.ClientWithResponses, error) {
	url := kibanaClient.URL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "api/fleet/"

	fleetClient, err := fleetapi.NewClientWithResponses(url, fleetapi.WithHTTPClient(kibanaClient.HTTP))
	if err != nil {
		return nil, err
	}
	return fleetClient, nil
}

// unexpectedResponse returns a diag.Diagnostics for when the Fleet API returns a
// response code that is unexpected. It will include the response body as detail.
func unexpectedResponse(statusCode int, body []byte) diag.Diagnostics {
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unexpected status code from server: got HTTP %d", statusCode),
			Detail:   string(body),
		},
	}
}

// ptrTo returns a pointer to the given value.
func ptrTo[T any](in T) *T {
	return &in
}
