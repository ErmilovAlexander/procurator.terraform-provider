package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SwitchesDataSource{}
var _ datasource.DataSourceWithConfigure = &SwitchesDataSource{}

type SwitchesDataSource struct{ client client.UmbraClient }

type switchesDataSourceModel struct {
	Switches []switchListItemModel `tfsdk:"switches"`
}

type switchListItemModel struct {
	ID       types.String     `tfsdk:"id"`
	MTU      types.Int64      `tfsdk:"mtu"`
	Networks types.List       `tfsdk:"networks"`
	State    types.String     `tfsdk:"state"`
	Errors   types.List       `tfsdk:"errors"`
	NICs     *switchNICsModel `tfsdk:"nics"`
}

func NewSwitchesDataSource() datasource.DataSource { return &SwitchesDataSource{} }

func (d *SwitchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_switches"
}

func (d *SwitchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*providerData)
	}
}

func (d *SwitchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"switches": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":       schema.StringAttribute{Computed: true},
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
			}},
		},
	}}
}

func (d *SwitchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data switchesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListSwitches(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list switches", err.Error())
		return
	}

	data.Switches = nil
	for _, sw := range items {
		item := switchListItemModel{
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
		data.Switches = append(data.Switches, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
