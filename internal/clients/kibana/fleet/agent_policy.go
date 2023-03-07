package fleet

import (
	"context"
	"net/http"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet/fleetapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func CreateAgentPolicy(ctx context.Context, client *kibana.Client, policy fleetapi.NewAgentPolicy) (*fleetapi.AgentPolicy, diag.Diagnostics) {
	c, err := fleetClient(client)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	resp, err := c.CreateAgentPolicyWithResponse(ctx, policy)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return resp.JSON200.Item, nil
	default:
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
}

func ReadAgentPolicy(ctx context.Context, client *kibana.Client, policyID string) (*fleetapi.AgentPolicy, diag.Diagnostics) {
	c, err := fleetClient(client)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	resp, err := c.AgentPolicyInfoWithResponse(ctx, policyID)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Item, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
}

func DeleteAgentPolicy(ctx context.Context, client *kibana.Client, policyID string) diag.Diagnostics {
	c, err := fleetClient(client)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.DeleteAgentPolicyWithResponse(ctx, fleetapi.DeleteAgentPolicyJSONRequestBody{AgentPolicyId: policyID})
	if err != nil {
		return diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return nil
	default:
		return unexpectedResponse(resp.StatusCode(), resp.Body)
	}
}
