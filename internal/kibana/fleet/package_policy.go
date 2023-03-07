package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet/fleetapi"
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
						Description:  "JSON-encoded string containing input level variables.",
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringIsJSON,
						//DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
						//	tflog.Info(context.Background(), fmt.Sprintf("DiffSuppressFunc for %v, old=%v, new=%v", k, oldValue, newValue))
						//	return true
						//},
					},
					"stream": {
						Description: "Input level variables.",
						Type:        schema.TypeList,
						Required:    true,
						MinItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"data_stream": {
									Description: `Name of the data_stream within the integration (e.g. "log").`,
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
									Description:  "JSON-encoded string containing stream level variables.",
									Type:         schema.TypeString,
									Optional:     true,
									ValidateFunc: validation.StringIsJSON,
									//DiffSuppressFunc: utils.DiffJsonSuppress,
									DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
										log.Printf("[INFO] DiffSuppressFunc for %v, old=%v, new=%v, resource=%v", k, oldValue, newValue, d.Get(k))
										return false // retain the diff
									},
								},
								"compiled_stream": {
									Description: "JSON-encoded string containing final configuration for the stream.",
									Type:        schema.TypeString,
									Computed:    true,
								},
								"vars": {
									Description: "Contains the stream level variables used by Fleet. This is merged merged with the defaults from the package manifests.",
									Type:        schema.TypeString,
									Computed:    true,
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

		CreateContext: resourcePackagePolicyCreate,
		DeleteContext: resourcePackagePolicyDelete,
		ReadContext:   resourcePackagePolicyRead,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: packagePolicySchema,
	}
}

func resourcePackagePolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	var newPolicy fleetapi.PackagePolicyRequest

	if v, ok := d.GetOk("name"); ok {
		newPolicy.Name = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		newPolicy.Description = ptrTo(v.(string))
	}
	if v, ok := d.GetOk("namespace"); ok {
		newPolicy.Namespace = ptrTo(v.(string))
	}
	if v, ok := d.GetOk("package.0.name"); ok {
		newPolicy.Package.Name = v.(string)
	}
	if v, ok := d.GetOk("package.0.version"); ok {
		newPolicy.Package.Version = v.(string)
	}
	if v, ok := d.GetOk("agent_policy_id"); ok {
		newPolicy.PolicyId = v.(string)
	}
	if v, ok := d.GetOk("vars_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &newPolicy.Vars); err != nil {
			return diag.FromErr(err)
		}
	}
	if v, ok := d.GetOk("input"); ok {
		inputList := v.([]interface{})
		inputsMap := make(map[string]fleetapi.PackagePolicyRequestInput, len(inputList))
		for i, v := range inputList {
			inputMap := v.(map[string]any)

			var policyTemplate, inputType string
			var input fleetapi.PackagePolicyRequestInput
			if v, ok := inputMap["policy_template"].(string); ok {
				policyTemplate = v
			}
			if v, ok := inputMap["type"].(string); ok {
				inputType = v
			}
			if v, ok := inputMap["enabled"].(bool); ok {
				input.Enabled = ptrTo(v)
			}
			if v, ok := inputMap["vars_json"].(string); ok && v != "" {
				if err := json.Unmarshal([]byte(v), &input.Vars); err != nil {
					return diag.FromErr(fmt.Errorf("failed unmarshaling input.%d.vars_json: %w", i, err))
				}
			}

			streamList := inputMap["stream"].([]any)
			streams := make(map[string]fleetapi.PackagePolicyRequestInputStream, len(streamList))
			for j, v := range streamList {
				streamMap := v.(map[string]any)

				var stream fleetapi.PackagePolicyRequestInputStream
				var dataStream string
				if v, ok := streamMap["data_stream"].(string); ok {
					dataStream = v
				}
				if v, ok := streamMap["enabled"].(bool); ok {
					stream.Enabled = ptrTo(v)
				}
				if v, ok := streamMap["vars_json"].(string); ok && v != "" {
					if err := json.Unmarshal([]byte(v), &stream.Vars); err != nil {
						return diag.FromErr(fmt.Errorf("failed unmarshaling input.%d.stream.%d.vars_json: %w", i, j, err))
					}
				}

				streams[newPolicy.Package.Name+"."+dataStream] = stream
			}
			input.Streams = &streams

			inputsMap[policyTemplate+"-"+inputType] = input
		}

		newPolicy.Inputs = &inputsMap
	}

	tflog.Info(ctx, fmt.Sprintf("CreatePackagePolicy for client %#v", client.GetKibanaClient()))
	packagePolicy, diags := fleet.CreatePackagePolicy(ctx, client.GetKibanaClient(), newPolicy)
	if diags.HasError() {
		return diags
	}

	d.SetId(packagePolicy.Id)
	return resourcePackagePolicyRead(ctx, d, meta)
}

func resourcePackagePolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	packagePolicy, diags := fleet.ReadPackagePolicy(ctx, client.GetKibanaClient(), d.Id())
	if diags.HasError() {
		return diags
	}

	// Not found.
	if packagePolicy == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("name", packagePolicy.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", packagePolicy.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("namespace", packagePolicy.Namespace); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("package", []map[string]any{
		{
			"name":    packagePolicy.Package.Name,
			"version": packagePolicy.Package.Version,
		},
	}); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("agent_policy_id", packagePolicy.PolicyId); err != nil {
		return diag.FromErr(err)
	}

	before := d.Get("input.0.stream.0.vars_json").(string)
	tflog.Info(ctx, fmt.Sprintf("BEFORE: input.0.stream.0.vars_json=%v", before))

	if inputs, diags := packagePolicyInputsToMap(d, packagePolicy); diags.HasError() {
		return diags
	} else if err := d.Set("input", inputs); err != nil {
		return diag.FromErr(err)
	}

	after := d.Get("input.0.stream.0.vars_json").(string)
	tflog.Info(ctx, fmt.Sprintf("AFTER: input.0.stream.0.vars_json=%v", after))

	diff, err := filterUnspecifiedVarsJSONKeys(before, after)
	if err != nil {
		diag.FromErr(err)
	}
	tflog.Info(ctx, fmt.Sprintf("DIFF: input.0.stream.0.vars_json=%s", diff))

	return nil
}

func resourcePackagePolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diags := clients.NewApiClient(d, meta)
	if diags.HasError() {
		return diags
	}

	if diags := fleet.DeletePackagePolicy(ctx, client.GetKibanaClient(), d.Id()); diags.HasError() {
		return diags
	}

	d.SetId("")
	return nil
}

func packagePolicyInputsToMap(d *schema.ResourceData, packagePolicy *fleetapi.PackagePolicy) ([]map[string]any, diag.Diagnostics) {
	// NOTE: The order of the inputs and streams in the response may not match order
	// from the config or the request. The request actually uses a map, so we need to
	// align the values.
	//
	// The returned values take the shape of one input for each policy_template + input type
	// combination. Then one stream within that input for each data_stream supporting
	// that input type.

	var inputs []map[string]any
	for inputIdx, input := range packagePolicy.Inputs {
		if !input.Enabled {
			continue
		}
		var streams []map[string]any
		if input.Streams != nil {
			for streamIdx, stream := range *input.Streams {
				streamMap := map[string]any{}
				if stream.DataStream != nil && stream.DataStream.Dataset != nil {
					// The returned data_stream.dataset value is in the format of
					// <package_name>.<dataset>. We need to strip the <package_name> to
					// make this align to the "data_stream" name of this provider.
					if parts := strings.SplitN(*stream.DataStream.Dataset, ".", 2); len(parts) == 2 {
						streamMap["data_stream"] = parts[1]
					}
				}
				if stream.Enabled != nil {
					streamMap["enabled"] = *stream.Enabled
					if !*stream.Enabled {
						continue
					}
				}
				if stream.Vars != nil {
					vars, err := json.Marshal(*stream.Vars)
					if err != nil {
						return nil, diag.FromErr(err)
					}

					streamMap["vars"] = string(vars)

					flatVars := map[string]any{}
					for k, v := range *stream.Vars {
						obj := v.(map[string]any)
						if value, ok := obj["value"]; ok {
							flatVars[k] = value
						}
					}

					flatVarsJSON, err := json.Marshal(flatVars)
					if err != nil {
						return nil, diag.FromErr(err)
					}

					if configured, ok := d.GetOk(fmt.Sprintf("input.%d.stream.%d.vars_json", inputIdx, streamIdx)); ok {
						flatVarsJSON, err = filterUnspecifiedVarsJSONKeys(configured.(string), string(flatVarsJSON))
						if err != nil {
							return nil, diag.FromErr(err)
						}
					}

					// TODO: Should probably generate vars_json based on a subset of keys configured in the source.
					streamMap["vars_json"] = string(flatVarsJSON)
				}
				if stream.CompiledStream != nil {
					compiledStreamJSON, err := json.Marshal(*stream.Vars)
					if err != nil {
						return nil, diag.FromErr(err)
					}
					streamMap["compiled_stream"] = string(compiledStreamJSON)
				}

				streams = append(streams, streamMap)
			}
		}

		inputs = append(inputs, map[string]any{
			"type":            input.Type,
			"enabled":         input.Enabled,
			"policy_template": input.PolicyTemplate,
			"stream":          streams,
		})
	}

	return inputs, nil
}

func filterUnspecifiedVarsJSONKeys(old, new string) ([]byte, error) {
	var oldMap, newMap map[string]any
	if err := json.Unmarshal([]byte(old), &oldMap); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(new), &newMap); err != nil {
		return nil, err
	}

	for k := range newMap {
		if _, found := oldMap[k]; !found {
			delete(newMap, k)
		}
	}

	jsonNewMap, err := json.Marshal(newMap)
	if err != nil {
		return nil, err
	}

	return jsonNewMap, nil
}
