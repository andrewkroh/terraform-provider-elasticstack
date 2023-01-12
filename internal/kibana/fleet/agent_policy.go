package fleet

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/fleet"
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
			Description: "Namespace",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"description": {
			Description: "Description of the policy.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
	}

	return &schema.Resource{
		Description: "Creates a new Fleet agent policy.",

		CreateContext: resourceAgentPolicyPost,
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

	policy, diags := client.GetFleetClient().GetAgentPolicy(ctx, d.Id())
	if diags.HasError() {
		return diags
	}

	// Not found.
	if policy == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("name", policy.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("namespace", policy.Namespace); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", policy.Description); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAgentPolicyPost(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	params := fleet.CreateAgentPolicyJSONRequestBody{
		Name:        d.Get("name").(string),
		Namespace:   d.Get("namespace").(string),
		Description: ptrTo(d.Get("description").(string)),
	}

	policy, diags := client.GetFleetClient().PostAgentPolicy(ctx, params)
	if diags.HasError() {
		return diags
	}

	d.SetId(policy.Id)
	return resourceAgentPolicyRead(ctx, d, meta)
}

func resourceAgentPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	return client.GetFleetClient().DeleteAgentPolicy(ctx, d.Id())
}

func ptrTo[T any](in T) *T {
	return &in
}
