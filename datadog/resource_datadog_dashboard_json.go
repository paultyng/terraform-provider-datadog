package datadog

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-datadog/datadog/internal/utils"
)

var computedFields = []string{"id", "author_handle", "author_name", "created_at", "modified_at", "url"}

const path = "/api/v1/dashboard"

func resourceDatadogDashboardJSON() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a Datadog dashboard JSON resource. This can be used to create and manage Datadog dashboards using the JSON definition.",
		CreateContext: resourceDatadogDashboardJSONCreate,
		ReadContext:   resourceDatadogDashboardJSONRead,
		UpdateContext: resourceDatadogDashboardJSONUpdate,
		DeleteContext: resourceDatadogDashboardJSONDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"dashboard": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					// Remove computed fields when comparing diffs
					attrMap, _ := structure.ExpandJsonFromString(v.(string))
					for _, f := range computedFields {
						delete(attrMap, f)
					}
					res, _ := structure.FlattenJsonToString(attrMap)
					return res
				},
				Description: "The JSON formatted definition of the Dashboard.",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The URL of the dashboard.",
			},
		},
	}
}

func resourceDatadogDashboardJSONRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	datadogClientV1 := providerConf.DatadogClientV1
	authV1 := providerConf.AuthV1

	id := d.Id()

	respByte, httpResp, err := utils.SendRequest(authV1, datadogClientV1, "GET", path+"/"+id, nil)
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	respMap, err := utils.ConvertResponseByteToMap(respByte)
	if err != nil {
		return diag.FromErr(err)
	}

	return updateDashboardJSONState(d, respMap)
}

func resourceDatadogDashboardJSONCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	datadogClientV1 := providerConf.DatadogClientV1
	authV1 := providerConf.AuthV1

	dashboard := d.Get("dashboard")

	respByte, _, err := utils.SendRequest(authV1, datadogClientV1, "POST", path, dashboard)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, "error creating resource")
	}

	respMap, err := utils.ConvertResponseByteToMap(respByte)
	if err != nil {
		return diag.FromErr(err)
	}

	id, ok := respMap["id"]
	if !ok {
		return diag.FromErr(errors.New("error retrieving id from response"))
	}
	d.SetId(id.(string))

	return updateDashboardJSONState(d, respMap)
}

func resourceDatadogDashboardJSONUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	datadogClientV1 := providerConf.DatadogClientV1
	authV1 := providerConf.AuthV1

	dashboard := d.Get("dashboard")
	id := d.Id()

	respByte, _, err := utils.SendRequest(authV1, datadogClientV1, "PUT", path+"/"+id, dashboard)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, "error updating dashboard")
	}

	respMap, err := utils.ConvertResponseByteToMap(respByte)
	if err != nil {
		return diag.FromErr(err)
	}

	return updateDashboardJSONState(d, respMap)
}

func resourceDatadogDashboardJSONDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConf := meta.(*ProviderConfiguration)
	datadogClientV1 := providerConf.DatadogClientV1
	authV1 := providerConf.AuthV1

	id := d.Id()

	_, _, err := utils.SendRequest(authV1, datadogClientV1, "DELETE", path+"/"+id, nil)
	if err != nil {
		return utils.TranslateClientErrorDiag(err, "error deleting dashboard")
	}

	return nil
}

func updateDashboardJSONState(d *schema.ResourceData, dashboard map[string]interface{}) diag.Diagnostics {
	if v, ok := dashboard["url"]; ok {
		if err := d.Set("url", v.(string)); err != nil {
			return diag.FromErr(err)
		}
	}

	// Remove computed fields from the object
	for _, f := range computedFields {
		delete(dashboard, f)
	}

	dashboardString, err := structure.FlattenJsonToString(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("dashboard", dashboardString); err != nil {
		return diag.FromErr(err)
	}
	return nil
}