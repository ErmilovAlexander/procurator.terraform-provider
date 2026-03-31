package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SwitchDataSource{}
var _ datasource.DataSourceWithConfigure = &SwitchDataSource{}

type SwitchDataSource struct{ client client.UmbraClient }

type switchDataSourceModel struct {
	ID       types.String     `tfsdk:"id"`
	MTU      types.Int64      `tfsdk:"mtu"`
	NICs     *switchNICsModel `tfsdk:"nics"`
	Networks types.List       `tfsdk:"networks"`
	State    types.String     `tfsdk:"state"`
	Errors   types.List       `tfsdk:"errors"`
}

func NewSwitchDataSource() datasource.DataSource { return &SwitchDataSource{} }

func (d *SwitchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_switch"
}

func (d *SwitchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData)
}

func (d *SwitchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":       schema.StringAttribute{Required: true},
		"mtu":      schema.Int64Attribute{Computed: true},
		"networks": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"state":    schema.StringAttribute{Computed: true},
		"errors":   schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"nics": schema.SingleNestedAttribute{
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"active":    schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"standby":   schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"unused":    schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"connected": schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"inherit":   schema.BoolAttribute{Computed: true},
			},
		},
	}}
}

func (d *SwitchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data switchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sw, err := d.client.GetSwitch(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Switch not found", err.Error())
		return
	}

	data = switchDataSourceModel{
		ID:       stringValue(sw.ID),
		MTU:      int64Value(int64(sw.MTU)),
		Networks: stringList(ctx, sw.Networks),
		State:    stringValue(sw.State),
		Errors:   stringList(ctx, sw.Errors),
		NICs: &switchNICsModel{
			Active:    stringList(ctx, sw.NICs.Active),
			Standby:   stringList(ctx, sw.NICs.Standby),
			Unused:    stringList(ctx, sw.NICs.Unused),
			Connected: stringList(ctx, sw.NICs.Connected),
			Inherit:   boolValue(sw.NICs.Inherit),
		},
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
