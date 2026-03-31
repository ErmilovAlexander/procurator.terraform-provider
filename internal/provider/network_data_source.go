package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NetworkDataSource{}
var _ datasource.DataSourceWithConfigure = &NetworkDataSource{}

type NetworkDataSource struct{ client client.UmbraClient }

type networkDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	VLAN        types.Int64  `tfsdk:"vlan"`
	NetBridge   types.String `tfsdk:"net_bridge"`
	Kind        types.String `tfsdk:"kind"`
	SwitchID    types.String `tfsdk:"switch_id"`
	VmsCount    types.Int64  `tfsdk:"vms_count"`
	ActivePorts types.Int64  `tfsdk:"active_ports"`
	State       types.String `tfsdk:"state"`
}

func NewNetworkDataSource() datasource.DataSource { return &NetworkDataSource{} }
func (d *NetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}
func (d *NetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData)
}
func (d *NetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":           schema.StringAttribute{Optional: true, Computed: true},
		"name":         schema.StringAttribute{Optional: true, Computed: true},
		"vlan":         schema.Int64Attribute{Computed: true},
		"net_bridge":   schema.StringAttribute{Computed: true},
		"kind":         schema.StringAttribute{Computed: true},
		"switch_id":    schema.StringAttribute{Computed: true},
		"vms_count":    schema.Int64Attribute{Computed: true},
		"active_ports": schema.Int64Attribute{Computed: true},
		"state":        schema.StringAttribute{Computed: true},
	}}
}
func (d *NetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasID := !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""
	if !hasID && !hasName {
		resp.Diagnostics.AddError("Missing network selector", "set id or name")
		return
	}
	if hasID && hasName {
		resp.Diagnostics.AddError("Ambiguous network selector", "set only one of id or name")
		return
	}
	var (
		n   *client.Network
		err error
	)
	if hasID {
		n, err = d.client.GetNetworkByID(ctx, data.ID.ValueString())
	} else {
		n, err = d.client.GetNetworkByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Network not found", err.Error())
		return
	}
	data = networkDataSourceModel{ID: types.StringValue(n.ID), Name: types.StringValue(n.Name), VLAN: types.Int64Value(int64(n.VLAN)), NetBridge: types.StringValue(n.NetBridge), Kind: types.StringValue(n.Kind), SwitchID: types.StringValue(n.SwitchID), VmsCount: types.Int64Value(int64(n.VmsCount)), ActivePorts: types.Int64Value(int64(n.ActivePorts)), State: types.StringValue(n.State)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
