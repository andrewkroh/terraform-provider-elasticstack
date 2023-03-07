package fleet

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet/fleetapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func AgentPolicy() *schema.Resource {
	agentPolicySchema := map[string]*schema.Schema{
		"name": {
			Description: "Name of agent policy.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"namespace": {
			Description: "Default namespace for data streams.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"description": {
			Description: "Description of the policy and how it is used.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
	}

	return &schema.Resource{
		Description: "Creates a new Fleet agent policy.",

		CreateContext: resourceAgentPolicyCreate,
		DeleteContext: resourceAgentPolicyDelete,
		ReadContext:   resourceAgentPolicyRead,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: agentPolicySchema,
	}
}

func resourceAgentPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	agentPolicy, diags := fleet.ReadAgentPolicy(ctx, client.GetKibanaClient(), d.Id())
	if diags.HasError() {
		return diags
	}

	// Not found.
	if agentPolicy == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("name", agentPolicy.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("namespace", agentPolicy.Namespace); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", agentPolicy.Description); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAgentPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	newAgentPolicy := fleetapi.NewAgentPolicy{
		Name:        d.Get("name").(string),
		Namespace:   d.Get("namespace").(string),
		Description: ptrTo(d.Get("description").(string)),
	}

	agentPolicy, diags := fleet.CreateAgentPolicy(ctx, client.GetKibanaClient(), newAgentPolicy)
	if diags.HasError() {
		return diags
	}

	d.SetId(agentPolicy.Id)
	return resourceAgentPolicyRead(ctx, d, meta)
}

func resourceAgentPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	if diags := fleet.DeleteAgentPolicy(ctx, client.GetKibanaClient(), d.Id()); diags.HasError() {
		return diags
	}

	d.SetId("")
	return nil
}
