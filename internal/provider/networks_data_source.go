package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NetworksDataSource{}
var _ datasource.DataSourceWithConfigure = &NetworksDataSource{}

type NetworksDataSource struct{ client client.UmbraClient }

type networksDataSourceModel struct {
	Filter   types.String           `tfsdk:"filter"`
	Networks []networkListItemModel `tfsdk:"networks"`
}

type networkListItemModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VLAN      types.Int64  `tfsdk:"vlan"`
	NetBridge types.String `tfsdk:"net_bridge"`
	Kind      types.String `tfsdk:"kind"`
	State     types.String `tfsdk:"state"`
}

func NewNetworksDataSource() datasource.DataSource { return &NetworksDataSource{} }
func (d *NetworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}
func (d *NetworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*providerData)
	}
}
func (d *NetworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"filter": schema.StringAttribute{Optional: true},
		"networks": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":         schema.StringAttribute{Computed: true},
				"name":       schema.StringAttribute{Computed: true},
				"vlan":       schema.Int64Attribute{Computed: true},
				"net_bridge": schema.StringAttribute{Computed: true},
				"kind":       schema.StringAttribute{Computed: true},
				"state":      schema.StringAttribute{Computed: true},
			}},
		},
	}}
}
func (d *NetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.ListNetworks(ctx, data.Filter.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list networks", err.Error())
		return
	}
	data.Networks = nil
	for _, n := range items {
		data.Networks = append(data.Networks, networkListItemModel{ID: types.StringValue(n.ID), Name: types.StringValue(n.Name), VLAN: types.Int64Value(int64(n.VLAN)), NetBridge: types.StringValue(n.NetBridge), Kind: types.StringValue(n.Kind), State: types.StringValue(n.State)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
