package fleet

import (
	"context"
	"net/http"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet/fleetapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func CreatePackagePolicy(ctx context.Context, client *kibana.Client, policy fleetapi.PackagePolicyRequest) (*fleetapi.PackagePolicy, diag.Diagnostics) {
	c, err := fleetClient(client)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	resp, err := c.CreatePackagePolicyWithResponse(ctx, policy)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &resp.JSON200.Item, nil
	default:
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
}

func ReadPackagePolicy(ctx context.Context, client *kibana.Client, policyID string) (*fleetapi.PackagePolicy, diag.Diagnostics) {
	c, err := fleetClient(client)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	resp, err := c.GetPackagePolicyWithResponse(ctx, policyID)
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

func DeletePackagePolicy(ctx context.Context, client *kibana.Client, policyID string) diag.Diagnostics {
	c, err := fleetClient(client)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := c.DeletePackagePolicyWithResponse(
		ctx, policyID, &fleetapi.DeletePackagePolicyParams{Force: ptrTo(true)})
	if err != nil {
		return diag.FromErr(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	default:
		return unexpectedResponse(resp.StatusCode(), resp.Body)
	}
}
