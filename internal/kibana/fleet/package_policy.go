package fleet

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/fleet"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func PackagePolicy() *schema.Resource {
	packagePolicySchema := map[string]*schema.Schema{
		"name": {
			Description: "Name of the package policy.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"description": {
			Description: "Description of the package policy.",
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
		},
		"namespace": {
			Description: "Namespace of the package policy.",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "default",
			ForceNew:    true,
		},
		"package": {
			Description: "Integration package to configure.",
			Type:        schema.TypeList,
			Required:    true,
			MinItems:    1,
			MaxItems:    1,
			ForceNew:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Description: "Name of the package.",
						Type:        schema.TypeString,
						Required:    true,
					},
					"version": {
						Description: "Version of the package.",
						Type:        schema.TypeString,
						Required:    true,
					},
				},
			},
		},
		"agent_policy_id": {
			Description: "Agent policy to add this package to.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"vars_json": {
			Description:      "JSON-encoded string containing root level variables.",
			Type:             schema.TypeString,
			Optional:         true,
			ForceNew:         true,
			ValidateFunc:     validation.StringIsJSON,
			DiffSuppressFunc: utils.DiffJsonSuppress,
		},
		"input": {
			Description: "List of inputs to configure.",
			Type:        schema.TypeList,
			Required:    true,
			MinItems:    1,
			ForceNew:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"policy_template": {
						Description: "Name of the policy template containing the input (see the integration's manifest.yml).",
						Type:        schema.TypeString,
						Required:    true,
					},
					"type": {
						Description: "Input type.",
						Type:        schema.TypeString,
						Required:    true,
					},
					"enabled": {
						Description: "Enable the input.",
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     true,
					},
					"vars_json": {
						Description:      "JSON-encoded string containing input level variables.",
						Type:             schema.TypeString,
						Optional:         true,
						ValidateFunc:     validation.StringIsJSON,
						DiffSuppressFunc: utils.DiffJsonSuppress,
					},
					"stream": {
						Description: "Input level variables.",
						Type:        schema.TypeList,
						Required:    true,
						MinItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"data_stream": {
									Description: "Name of the data_stream within the integration.",
									Type:        schema.TypeString,
									Required:    true,
								},
								"enabled": {
									Description: "Enabled enable or disable that stream.",
									Type:        schema.TypeBool,
									Optional:    true,
									Default:     true,
								},
								"vars_json": {
									Description:      "JSON-encoded string containing stream level variables.",
									Type:             schema.TypeString,
									Optional:         true,
									ValidateFunc:     validation.StringIsJSON,
									DiffSuppressFunc: utils.DiffJsonSuppress,
								},
							},
						},
					},
				},
			},
		},
	}

	return &schema.Resource{
		Description: "Creates a new Fleet package policy.",

		CreateContext: resourcePackagePolicyPost,
		DeleteContext: resourcePackagePolicyDelete,
		ReadContext:   resourcePackagePolicyRead,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: packagePolicySchema,
	}
}

func resourcePackagePolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "resourcePackagePolicyRead")

	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	policy, diags := client.GetFleetClient().ReadPackagePolicy(ctx, d.Id())
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
	if err := d.Set("package.0.name", policy.Package.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("package.0.version", policy.Package.Version); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("agent_policy_id", policy.PolicyId); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePackagePolicyPost(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	defer func() {
		if r := recover(); r != nil {
			tflog.Error(ctx, fmt.Sprintln("Recovered in f", r))
		}
	}()
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	params := fleet.CreatePackagePolicyJSONRequestBody{
		Name:        d.Get("name").(string),
		Namespace:   ptrTo(d.Get("namespace").(string)),
		Description: ptrTo(d.Get("description").(string)),
		PolicyId:    d.Get("agent_policy_id").(string),
	}
	params.Package.Name = d.Get("package.0.name").(string)
	params.Package.Version = d.Get("package.0.version").(string)

	//if v, ok := d.GetOk("vars_json"); ok {
	//	var vars map[string]any
	//	if err := json.NewDecoder(strings.NewReader(v.(string))).Decode(&vars); err != nil {
	//		return diag.FromErr(err)
	//	}
	//	if vars != nil {
	//		params.Vars = &vars
	//	}
	//}

	inputs := map[string]fleet.PackagePolicyInputSimplified{}
	for _, item := range d.Get("input").([]any) {
		inputMap := item.(map[string]any)
		policyTemplate := inputMap["policy_template"].(string)
		inputType := inputMap["type"].(string)
		enabled := inputMap["enabled"].(bool)

		//vars := map[string]any{}
		//if v, ok := inputMap["vars_json"].(string); ok {
		//	if err := json.NewDecoder(strings.NewReader(v)).Decode(&vars); err != nil {
		//		return diag.FromErr(err)
		//	}
		//}

		streams := map[string]fleet.PackagePolicyStreamSimplified{}
		for _, item := range inputMap["stream"].([]any) {
			streamMap := item.(map[string]any)
			streamDataStream := streamMap["data_stream"].(string)
			streamEnabled := streamMap["enabled"].(bool)

			//streamVars := map[string]any{}
			//if v, ok := streamMap["vars_json"].(string); ok {
			//	if err := json.NewDecoder(strings.NewReader(v)).Decode(&streamVars); err != nil {
			//		return diag.FromErr(err)
			//	}
			//}

			streams[params.Package.Name+"."+streamDataStream] = fleet.PackagePolicyStreamSimplified{
				Enabled: ptrTo(streamEnabled),
				//Vars:    &streamVars,
			}
		}
		inputs[policyTemplate+"-"+inputType] = fleet.PackagePolicyInputSimplified{
			Enabled: ptrTo(enabled),
			//Vars:    &vars,
			Streams: &streams,
		}
	}
	params.Inputs = &inputs

	tflog.Debug(ctx, fmt.Sprintf("Inputs data type %v", spew.Sdump(params)))

	policy, diags := client.GetFleetClient().CreatePackagePolicy(ctx, params)
	if diags.HasError() {
		return diags
	}

	d.SetId(policy.Id)
	return resourcePackagePolicyRead(ctx, d, meta)
}

func resourcePackagePolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	return client.GetFleetClient().DeletePackagePolicy(ctx, d.Id())
}
