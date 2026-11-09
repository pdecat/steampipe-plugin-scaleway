package scaleway

import (
	"context"

	"github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/turbot/steampipe-plugin-sdk/v6/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v6/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v6/plugin/transform"
)

//// TABLE DEFINITION

func tableScalewayIPAMIP(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:              "scaleway_ipam_ip",
		Description:       "An IPAM IP is an IP address reserved by Scaleway IP Address Manager, such as the private IP of an Instance on a Private Network.",
		GetMatrixItemFunc: BuildRegionList,
		List: &plugin.ListConfig{
			Hydrate: listIPAMIPs,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:    "region",
					Require: plugin.Optional,
				},
				{
					Name:    "project",
					Require: plugin.Optional,
				},
				{
					Name:    "private_network_id",
					Require: plugin.Optional,
				},
				{
					Name:    "resource_id",
					Require: plugin.Optional,
				},
				{
					Name:    "resource_type",
					Require: plugin.Optional,
				},
				{
					Name:    "is_ipv6",
					Require: plugin.Optional,
				},
			},
		},
		Get: &plugin.GetConfig{
			Hydrate:    getIPAMIP,
			KeyColumns: plugin.AllColumns([]string{"id", "region"}),
		},
		Columns: []*plugin.Column{
			{
				Name:        "id",
				Description: "An unique identifier of the IP.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ID"),
			},
			{
				Name:        "address",
				Description: "The IPv4 or IPv6 address, in CIDR notation.",
				Type:        proto.ColumnType_INET,
				Transform:   transform.FromField("Address").Transform(ipNetToString),
			},
			{
				Name:        "is_ipv6",
				Description: "True if the IP is an IPv6 address, false if it is an IPv4 address.",
				Type:        proto.ColumnType_BOOL,
				Transform:   transform.FromField("IsIPv6"),
			},
			{
				Name:        "created_at",
				Description: "The time when the IP was reserved.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "updated_at",
				Description: "The time when the IP was last modified.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "private_network_id",
				Description: "The ID of the Private Network the IP was reserved in, if it was reserved in one.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Source.PrivateNetworkID"),
			},
			{
				Name:        "subnet_id",
				Description: "The ID of the subnet the IP was reserved in, if it was reserved in one.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Source.SubnetID"),
			},
			{
				Name:        "source",
				Description: "The source pool the IP was reserved in: a zone, a region, a Private Network, a subnet or a VPC.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "resource_id",
				Description: "The ID of the resource the IP is attached to, such as the private NIC of an Instance.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Resource.ID"),
			},
			{
				Name:        "resource_type",
				Description: "The type of the resource the IP is attached to, such as instance_private_nic.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Resource.Type"),
			},
			{
				Name:        "resource_name",
				Description: "The name of the resource the IP is attached to, which resolves to the IP in the Private Network DNS.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Resource.Name"),
			},
			{
				Name:        "resource_mac_address",
				Description: "The MAC address of the resource the IP is attached to.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Resource.MacAddress"),
			},
			{
				Name:        "resource",
				Description: "The resource the IP is attached to.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "reverses",
				Description: "A list of reverse DNS records associated with the IP.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "tags",
				Description: "A list of tags associated with the IP.",
				Type:        proto.ColumnType_JSON,
			},

			// Scaleway standard columns
			{
				Name:        "region",
				Description: "Specifies the region where the IP resides.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Region").Transform(transform.ToString),
			},
			{
				Name:        "zone",
				Description: "Specifies the zone of the IP, if it is zonal.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Zone"),
			},
			{
				Name:        "project",
				Description: "The ID of the project where the IP resides.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ProjectID"),
			},

			// Steampipe standard columns
			{
				Name:        "title",
				Description: "Title of the resource.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Address").Transform(ipNetToString),
			},
		},
	}
}

//// LIST FUNCTION

func listIPAMIPs(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.EqualsQualString("region")

	parseRegionData, err := scw.ParseRegion(region)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_ipam_ip.listIPAMIPs", "region_parsing_error", err)
		return nil, err
	}

	quals := d.EqualsQuals
	if quals["region"] != nil && quals["region"].GetStringValue() != region {
		return nil, nil
	}

	// Create client
	client, err := getSessionConfig(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_ipam_ip.listIPAMIPs", "connection_error", err)
		return nil, err
	}

	// Create SDK objects for Scaleway IPAM product
	ipamApi := ipam.NewAPI(client)

	req := &ipam.ListIPsRequest{
		Region: parseRegionData,
		Page:   scw.Int32Ptr(1),
	}
	// Additional filters
	if quals["project"] != nil {
		req.ProjectID = scw.StringPtr(quals["project"].GetStringValue())
	}
	if quals["private_network_id"] != nil {
		req.PrivateNetworkID = scw.StringPtr(quals["private_network_id"].GetStringValue())
	}
	if quals["resource_id"] != nil {
		req.ResourceID = scw.StringPtr(quals["resource_id"].GetStringValue())
	}
	if quals["resource_type"] != nil {
		req.ResourceType = ipam.ResourceType(quals["resource_type"].GetStringValue())
	}
	if quals["is_ipv6"] != nil {
		req.IsIPv6 = scw.BoolPtr(quals["is_ipv6"].GetBoolValue())
	}

	// Retrieve the list of IPs
	maxResult := int64(100)

	// Reduce the basic request limit down if the user has only requested a small number of rows
	limit := d.QueryContext.Limit
	if d.QueryContext.Limit != nil {
		if *limit < maxResult {
			maxResult = *limit
		}
	}
	req.PageSize = scw.Uint32Ptr(uint32(maxResult))

	var count int

	for {
		resp, err := ipamApi.ListIPs(req)
		if err != nil {
			plugin.Logger(ctx).Error("scaleway_ipam_ip.listIPAMIPs", "query_error", err)
			return nil, err
		}

		for _, ip := range resp.IPs {
			d.StreamListItem(ctx, ip)

			// Increase the resource count by 1
			count++

			// Context can be cancelled due to manual cancellation or the limit has been hit
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}

		// Stop when the last page has been read, or when the API returns an empty
		// page, which would otherwise loop forever if resources went away mid-listing
		if len(resp.IPs) == 0 || uint64(count) >= resp.TotalCount {
			break
		}
		req.Page = scw.Int32Ptr(*req.Page + 1)
	}

	return nil, nil
}

//// HYDRATE FUNCTIONS

func getIPAMIP(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.EqualsQualString("region")

	parseRegionData, err := scw.ParseRegion(region)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_ipam_ip.getIPAMIP", "region_parsing_error", err)
		return nil, err
	}

	if d.EqualsQuals["region"].GetStringValue() != region {
		return nil, nil
	}

	// Create client
	client, err := getSessionConfig(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_ipam_ip.getIPAMIP", "connection_error", err)
		return nil, err
	}

	// Create SDK objects for Scaleway IPAM product
	ipamApi := ipam.NewAPI(client)

	id := d.EqualsQuals["id"].GetStringValue()

	// No inputs
	if id == "" || region == "" {
		return nil, nil
	}

	data, err := ipamApi.GetIP(&ipam.GetIPRequest{
		IPID:   id,
		Region: parseRegionData,
	})
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_ipam_ip.getIPAMIP", "query_error", err)
		if is404Error(err) {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}

//// TRANSFORM FUNCTIONS

// ipNetToString renders a scw.IPNet in CIDR notation. Its String method comes
// from the embedded net.IPNet and has a pointer receiver, so the SDK's generic
// string conversion cannot reach it and would return an empty value instead.
func ipNetToString(_ context.Context, d *transform.TransformData) (interface{}, error) {
	var ipNet *scw.IPNet
	switch v := d.Value.(type) {
	case scw.IPNet:
		ipNet = &v
	case *scw.IPNet:
		ipNet = v
	}
	if ipNet == nil || ipNet.IP == nil {
		return nil, nil
	}
	return ipNet.String(), nil
}
