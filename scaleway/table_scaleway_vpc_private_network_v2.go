package scaleway

import (
	"context"

	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/turbot/steampipe-plugin-sdk/v6/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v6/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v6/plugin/transform"
)

//// TABLE DEFINITION

func tableScalewayVPCPrivateNetworkV2(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:              "scaleway_vpc_private_network_v2",
		Description:       "A VPC private network allows interconnecting your instances in an isolated and private network. Private networks are regional resources exposed by the VPC v2 API.",
		GetMatrixItemFunc: BuildRegionList,
		List: &plugin.ListConfig{
			Hydrate: listVPCPrivateNetworksV2,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:    "name",
					Require: plugin.Optional,
				},
				{
					Name:    "region",
					Require: plugin.Optional,
				},
			},
		},
		Get: &plugin.GetConfig{
			Hydrate:    getVPCPrivateNetworkV2,
			KeyColumns: plugin.AllColumns([]string{"id", "region"}),
		},
		Columns: []*plugin.Column{
			{
				Name:        "name",
				Description: "The user-defined name of the private network.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "id",
				Description: "An unique identifier of the private network.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ID"),
			},
			{
				Name:        "vpc_id",
				Description: "The VPC in which the private network is available.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("VpcID"),
			},
			{
				Name:        "created_at",
				Description: "The time when the private network was created.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "updated_at",
				Description: "The time when the private network was last updated.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "tags",
				Description: "A list of tags associated with the private network.",
				Type:        proto.ColumnType_JSON,
			},

			// Scaleway standard columns
			{
				Name:        "region",
				Description: "Specifies the region where the private network resides.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Region").Transform(transform.ToString),
			},
			{
				Name:        "project",
				Description: "The ID of the project where the private network resides.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ProjectID"),
			},
			{
				Name:        "organization",
				Description: "The ID of the organization where the private network resides.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("OrganizationID"),
			},

			// Steampipe standard columns
			{
				Name:        "title",
				Description: "Title of the resource.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Name"),
			},
		},
	}
}

//// LIST FUNCTION

func listVPCPrivateNetworksV2(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.EqualsQualString("region")

	parseRegionData, err := scw.ParseRegion(region)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.listVPCPrivateNetworksV2", "region_parsing_error", err)
		return nil, err
	}

	quals := d.EqualsQuals
	if quals["region"] != nil && quals["region"].GetStringValue() != region {
		return nil, nil
	}

	// Create client
	client, err := getSessionConfig(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.listVPCPrivateNetworksV2", "connection_error", err)
		return nil, err
	}

	// Create SDK objects for Scaleway VPC product
	vpcApi := vpc.NewAPI(client)

	req := &vpc.ListPrivateNetworksRequest{
		Region: parseRegionData,
		Page:   scw.Int32Ptr(1),
	}
	// Additional filters
	if quals["name"] != nil {
		req.Name = scw.StringPtr(quals["name"].GetStringValue())
	}

	// Retrieve the list of private networks
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
		resp, err := vpcApi.ListPrivateNetworks(req)
		if err != nil {
			plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.listVPCPrivateNetworksV2", "query_error", err)
			return nil, err
		}

		for _, network := range resp.PrivateNetworks {
			d.StreamListItem(ctx, network)

			// Increase the resource count by 1
			count++

			// Context can be cancelled due to manual cancellation or the limit has been hit
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}

		if resp.TotalCount == uint32(count) {
			break
		}
		req.Page = scw.Int32Ptr(*req.Page + 1)

	}

	return nil, nil
}

//// HYDRATE FUNCTIONS

func getVPCPrivateNetworkV2(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	region := d.EqualsQualString("region")

	parseRegionData, err := scw.ParseRegion(region)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.getVPCPrivateNetworkV2", "region_parsing_error", err)
		return nil, err
	}

	if d.EqualsQuals["region"].GetStringValue() != region {
		return nil, nil
	}

	// Create client
	client, err := getSessionConfig(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.getVPCPrivateNetworkV2", "connection_error", err)
		return nil, err
	}

	// Create SDK objects for Scaleway VPC product
	vpcApi := vpc.NewAPI(client)

	id := d.EqualsQuals["id"].GetStringValue()
	vpcRegion := d.EqualsQuals["region"].GetStringValue()

	// No inputs
	if id == "" && vpcRegion == "" {
		return nil, nil
	}

	data, err := vpcApi.GetPrivateNetwork(&vpc.GetPrivateNetworkRequest{
		PrivateNetworkID: id,
		Region:           parseRegionData,
	})
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_vpc_private_network_v2.getVPCPrivateNetworkV2", "query_error", err)
		if is404Error(err) {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}
