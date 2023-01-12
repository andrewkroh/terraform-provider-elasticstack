package fleet

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/fleet"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func EnrollmentAPIKey() *schema.Resource {
	enrollmentApiKeySchema := map[string]*schema.Schema{
		"policy_id": {
			Description: "Identifier for the stored script. Must be unique within the cluster.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"name": {
			Description: "Script language. For search templates, use `mustache`.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"api_key": {
			Description: "API key",
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
		},
		"api_key_id": {
			Description: "API key ID",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"active": {
			Description: "Is API key active?",
			Type:        schema.TypeBool,
			Computed:    true,
		},
	}

	return &schema.Resource{
		Description: "Creates a Fleet enrollment token.",

		CreateContext: resourceEnrollmentAPIKeyPost,
		DeleteContext: resourceEnrollmentAPIKeyDelete,
		ReadContext:   resourceEnrollmentAPIKeyRead,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: enrollmentApiKeySchema,
	}
}

func resourceEnrollmentAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	key, diags := client.GetFleetClient().GetEnrollmentApiKey(ctx, d.Id())
	if diags.HasError() {
		return diags
	}

	// Not found.
	if key == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("api_key", key.ApiKey); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("api_key_id", key.ApiKeyId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("active", key.Active); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceEnrollmentAPIKeyPost(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	params := fleet.CreateEnrollmentApiKeysJSONRequestBody{
		Name:     d.Get("name").(string),
		PolicyId: d.Get("policy_id").(string),
	}

	key, diags := client.GetFleetClient().PostEnrollmentAPIKeys(ctx, params)
	if diags.HasError() {
		return diags
	}

	d.SetId(key.Id)
	return resourceEnrollmentAPIKeyRead(ctx, d, meta)
}

func resourceEnrollmentAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	return client.GetFleetClient().DeleteEnrollmentApiKey(ctx, d.Id())
}
