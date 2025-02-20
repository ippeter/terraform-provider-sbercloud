package vpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chnsz/golangsdk"
	"github.com/chnsz/golangsdk/openstack/networking/v1/routetables"
	"github.com/chnsz/golangsdk/openstack/networking/v2/routes"
	"github.com/chnsz/golangsdk/pagination"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/config"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/utils"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/utils/fmtp"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/utils/logp"
)

func ResourceVPCRouteTableRoute() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVpcRTBRouteCreate,
		ReadContext:   resourceVpcRTBRouteRead,
		UpdateContext: resourceVpcRTBRouteUpdate,
		DeleteContext: resourceVpcRTBRouteDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceVpcRTBRouteImportState,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: utils.ValidateCIDR,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"nexthop": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"route_table_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"route_table_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceVpcRTBRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*config.Config)
	vpcClient, err := config.NetworkingV1Client(config.GetRegion(d))
	if err != nil {
		return fmtp.DiagErrorf("Error creating VPC client: %s", err)
	}

	var routeTableID string
	vpcID := d.Get("vpc_id").(string)
	if v, ok := d.GetOk("route_table_id"); ok {
		routeTableID = v.(string)
	} else {
		routeTableID, err = getDefaultRouteTable(vpcClient, vpcID)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	destination := d.Get("destination").(string)
	desc := d.Get("description").(string)
	routeOpts := routetables.RouteOpts{
		Type:        d.Get("type").(string),
		NextHop:     d.Get("nexthop").(string),
		Destination: destination,
		Description: &desc,
	}

	updateOpts := routetables.UpdateOpts{
		Routes: map[string][]routetables.RouteOpts{
			"add": {routeOpts},
		},
	}

	logp.Printf("[DEBUG] add route in VPC route table[%s]: %#v", routeTableID, updateOpts)
	_, err = routetables.Update(vpcClient, routeTableID, updateOpts).Extract()
	if err != nil {
		return fmtp.DiagErrorf("Error creating VPC route: %s", err)
	}

	routeID := fmt.Sprintf("%s/%s", routeTableID, destination)
	d.SetId(routeID)
	return resourceVpcRTBRouteRead(ctx, d, meta)

}

func resourceVpcRTBRouteRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*config.Config)
	region := config.GetRegion(d)
	vpcClient, err := config.NetworkingV1Client(region)
	if err != nil {
		return fmtp.DiagErrorf("Error creating VPC client: %s", err)
	}

	var diags diag.Diagnostics
	routeTableID, destination := parseResourceID(d.Id())

	// Compatible with previous versions: conver ID to new format
	if routeTableID == "" {
		oldID := d.Id()
		logp.Printf("[WARN] The resource ID %s is in the old format, try to upgrade it to the new format", oldID)

		newID, subDiags := convertRouteIDtoNewFormat(d, config, oldID)
		if subDiags.HasError() {
			return subDiags
		}

		diags = subDiags
		d.SetId(newID)
		logp.Printf("[DEBUG] The resource ID %s has upgraded to %s", oldID, d.Id())
		routeTableID, destination = parseResourceID(newID)
	}

	routeTable, err := routetables.Get(vpcClient, routeTableID).Extract()
	if err != nil {
		diags = append(diags, fmtp.DiagErrorf("Error retrieving VPC route table %s: %s", routeTableID, err)[0])
		return diags
	}

	var route *routetables.Route
	for _, item := range routeTable.Routes {
		if item.DestinationCIDR == destination {
			route = &item
			break
		}
	}

	if route == nil {
		diags = append(diags, fmtp.DiagErrorf("can not find the vpc route %s with %s", routeTableID, destination)[0])
		return diags
	}

	mErr := multierror.Append(nil,
		d.Set("region", region),
		d.Set("type", route.Type),
		d.Set("nexthop", route.NextHop),
		d.Set("destination", route.DestinationCIDR),
		d.Set("description", route.Description),
		d.Set("vpc_id", routeTable.VpcID),
		d.Set("route_table_id", routeTable.ID),
		d.Set("route_table_name", routeTable.Name),
	)

	if err := mErr.ErrorOrNil(); err != nil {
		diags = append(diags, fmtp.DiagErrorf("Error saving VPC route: %s", err)[0])
	}

	return diags
}

func resourceVpcRTBRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*config.Config)
	vpcClient, err := config.NetworkingV1Client(config.GetRegion(d))
	if err != nil {
		return fmtp.DiagErrorf("Error creating VPC client: %s", err)
	}

	routeTableID, _ := parseResourceID(d.Id())
	desc := d.Get("description").(string)
	routeOpts := routetables.RouteOpts{
		Type:        d.Get("type").(string),
		NextHop:     d.Get("nexthop").(string),
		Destination: d.Get("destination").(string),
		Description: &desc,
	}

	updateOpts := routetables.UpdateOpts{
		Routes: map[string][]routetables.RouteOpts{
			"mod": {routeOpts},
		},
	}

	logp.Printf("[DEBUG] update route in vpc route table[%s]: %#v", routeTableID, updateOpts)
	if _, err := routetables.Update(vpcClient, routeTableID, updateOpts).Extract(); err != nil {
		return fmtp.DiagErrorf("Error updating VPC route: %s", err)
	}

	return resourceVpcRTBRouteRead(ctx, d, meta)
}

func resourceVpcRTBRouteDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*config.Config)
	vpcClient, err := config.NetworkingV1Client(config.GetRegion(d))
	if err != nil {
		return fmtp.DiagErrorf("Error creating VPC client: %s", err)
	}

	routeTableID, _ := parseResourceID(d.Id())
	routeOpts := routetables.RouteOpts{
		Destination: d.Get("destination").(string),
		Type:        d.Get("type").(string),
		NextHop:     d.Get("nexthop").(string),
	}

	updateOpts := routetables.UpdateOpts{
		Routes: map[string][]routetables.RouteOpts{
			"del": {routeOpts},
		},
	}

	logp.Printf("[DEBUG] delete route in vpc route table[%s]: %#v", routeTableID, updateOpts)
	if _, err := routetables.Update(vpcClient, routeTableID, updateOpts).Extract(); err != nil {
		return fmtp.DiagErrorf("Error deleting VPC route: %s", err)
	}

	d.SetId("")
	return nil
}

func getDefaultRouteTable(client *golangsdk.ServiceClient, vpcID string) (string, error) {
	listOpts := routetables.ListOpts{
		VpcID: vpcID,
	}
	pager := routetables.List(client, &listOpts)

	defaultID := ""
	err := pager.EachPage(func(page pagination.Page) (b bool, err error) {
		tableList, err := routetables.ExtractRouteTables(page)
		if err != nil {
			return false, err
		}
		for _, table := range tableList {
			if table.Default {
				// find the default route table, stop iterating
				defaultID = table.ID
				return false, nil
			}
		}
		return true, nil
	})

	if err != nil {
		return "", err
	}
	return defaultID, nil
}

func parseResourceID(id string) (string, string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}

func resourceVpcRTBRouteImportState(_ context.Context, d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {

	routeID, _ := parseResourceID(d.Id())
	if routeID == "" {
		return nil, fmt.Errorf("Invalid format specified for import id, must be <route_table_id>/<destination>")
	}

	return []*schema.ResourceData{d}, nil
}

func convertRouteIDtoNewFormat(d *schema.ResourceData, conf *config.Config, oldID string) (string, diag.Diagnostics) {
	var diags = diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Deprecated ID format",
			Detail:   fmt.Sprintf("The resource ID %s is in the old format, try to upgrade it to the new format", oldID),
		},
	}

	region := conf.GetRegion(d)
	vpcClient, err := conf.NetworkingV1Client(region)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error creating VPC client",
			Detail:   err.Error(),
		})
		return "", diags
	}
	networkClient, err := conf.NetworkingV2Client(region)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error creating network client",
			Detail:   err.Error(),
		})
		return "", diags
	}

	if _, err := routes.Get(networkClient, oldID).Extract(); err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("The resource %s does not exist", oldID),
			})
			d.SetId("")
		} else {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Error retrieving VPC route",
				Detail:   err.Error(),
			})
		}

		return "", diags
	}

	destination := d.Get("destination").(string)
	rtbID := d.Get("route_table_id").(string)
	if destination == "" || rtbID != "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "can not get the destination or the route_table_id is not empty",
		})
		return "", diags
	}

	vpcID := d.Get("vpc_id").(string)
	routeTableID, err := getDefaultRouteTable(vpcClient, vpcID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "failed to get the default route table",
			Detail:   err.Error(),
		})
		return "", diags
	}

	newRouteID := fmt.Sprintf("%s/%s", routeTableID, destination)
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("The resource ID is upgraded to %s", newRouteID),
	})
	return newRouteID, diags
}
