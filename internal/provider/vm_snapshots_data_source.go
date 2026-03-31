package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VMSnapshotsDataSource{}
var _ datasource.DataSourceWithConfigure = &VMSnapshotsDataSource{}

type VMSnapshotsDataSource struct{ client client.Client }

type vmSnapshotsDataSourceModel struct {
	VMID      types.String          `tfsdk:"vm_id"`
	CurrentID types.Int64           `tfsdk:"current_id"`
	Items     []vmSnapshotListModel `tfsdk:"items"`
}

type vmSnapshotListModel struct {
	SnapshotID types.Int64  `tfsdk:"snapshot_id"`
	Name       types.String `tfsdk:"name"`
	Timestamp  types.Int64  `tfsdk:"timestamp"`
	Size       types.Int64  `tfsdk:"size"`
	Current    types.Bool   `tfsdk:"current"`
}

func NewVMSnapshotsDataSource() datasource.DataSource { return &VMSnapshotsDataSource{} }
func (d *VMSnapshotsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_snapshots"
}
func (d *VMSnapshotsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData).client
}
func (d *VMSnapshotsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"vm_id":      schema.StringAttribute{Required: true},
		"current_id": schema.Int64Attribute{Computed: true},
	}, Blocks: map[string]schema.Block{
		"items": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"snapshot_id": schema.Int64Attribute{Computed: true},
			"name":        schema.StringAttribute{Computed: true},
			"timestamp":   schema.Int64Attribute{Computed: true},
			"size":        schema.Int64Attribute{Computed: true},
			"current":     schema.BoolAttribute{Computed: true},
		}}},
	}}
}
func (d *VMSnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vmSnapshotsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	snaps, currentID, err := d.client.ListVMSnapshots(ctx, state.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM snapshots", err.Error())
		return
	}
	state.CurrentID = types.Int64Value(currentID)
	state.Items = nil
	for _, s := range snaps {
		state.Items = append(state.Items, vmSnapshotListModel{SnapshotID: types.Int64Value(s.ID), Name: stringValue(s.Name), Timestamp: types.Int64Value(s.Timestamp), Size: types.Int64Value(s.Size), Current: boolValue(s.ID == currentID)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
