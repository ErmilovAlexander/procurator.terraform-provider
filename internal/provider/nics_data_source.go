package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NICsDataSource{}
var _ datasource.DataSourceWithConfigure = &NICsDataSource{}

type NICsDataSource struct{ client client.UmbraClient }

type nicsDataSourceModel struct {
	Filter types.String   `tfsdk:"filter"`
	NICs   []nicItemModel `tfsdk:"nics"`
}

type nicItemModel struct {
	ID       types.String `tfsdk:"id"`
	Adapter  types.String `tfsdk:"adapter"`
	PCIAddr  types.String `tfsdk:"pci_addr"`
	Driver   types.String `tfsdk:"driver"`
	Carrier  types.Bool   `tfsdk:"carrier"`
	Speed    types.Int64  `tfsdk:"speed"`
	Duplex   types.String `tfsdk:"duplex"`
	Networks types.List   `tfsdk:"networks"`
	Sriov    types.Bool   `tfsdk:"sr_iov"`
	CDP      types.Bool   `tfsdk:"cdp"`
	LLDP     types.Bool   `tfsdk:"lldp"`
	Managed  types.Bool   `tfsdk:"managed"`
	Name     types.String `tfsdk:"name"`
	SwitchID types.String `tfsdk:"switch_id"`
	MAC      types.String `tfsdk:"mac"`
	State    types.String `tfsdk:"state"`
	Errors   types.List   `tfsdk:"errors"`
}

func NewNICsDataSource() datasource.DataSource { return &NICsDataSource{} }

func (d *NICsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nics"
}

func (d *NICsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*providerData)
	}
}

func (d *NICsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"filter": schema.StringAttribute{Optional: true},
		"nics": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":        schema.StringAttribute{Computed: true},
				"adapter":   schema.StringAttribute{Computed: true},
				"pci_addr":  schema.StringAttribute{Computed: true},
				"driver":    schema.StringAttribute{Computed: true},
				"carrier":   schema.BoolAttribute{Computed: true},
				"speed":     schema.Int64Attribute{Computed: true},
				"duplex":    schema.StringAttribute{Computed: true},
				"networks":  schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"sr_iov":    schema.BoolAttribute{Computed: true},
				"cdp":       schema.BoolAttribute{Computed: true},
				"lldp":      schema.BoolAttribute{Computed: true},
				"managed":   schema.BoolAttribute{Computed: true},
				"name":      schema.StringAttribute{Computed: true},
				"switch_id": schema.StringAttribute{Computed: true},
				"mac":       schema.StringAttribute{Computed: true},
				"state":     schema.StringAttribute{Computed: true},
				"errors":    schema.ListAttribute{Computed: true, ElementType: types.StringType},
			}},
		},
	}}
}

func (d *NICsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data nicsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter := ""
	if !data.Filter.IsNull() && !data.Filter.IsUnknown() {
		filter = data.Filter.ValueString()
	}
	items, err := d.client.ListNICs(ctx, filter)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list NICs", err.Error())
		return
	}

	data.NICs = nil
	for _, nic := range items {
		data.NICs = append(data.NICs, nicItemModel{
			ID:       stringValue(nic.ID),
			Adapter:  stringValue(nic.Adapter),
			PCIAddr:  stringValue(nic.PCIAddr),
			Driver:   stringValue(nic.Driver),
			Carrier:  boolValue(nic.Carrier),
			Speed:    int64Value(int64(nic.Speed)),
			Duplex:   stringValue(nic.Duplex),
			Networks: stringList(ctx, nic.Networks),
			Sriov:    boolValue(nic.Sriov),
			CDP:      boolValue(nic.CDP),
			LLDP:     boolValue(nic.LLDP),
			Managed:  boolValue(nic.Managed),
			Name:     stringValue(nic.Name),
			SwitchID: stringValue(nic.SwitchID),
			MAC:      stringValue(nic.MAC),
			State:    stringValue(nic.State),
			Errors:   stringList(ctx, nic.Errors),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
