package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HostDataSource{}
var _ datasource.DataSourceWithConfigure = &HostDataSource{}

type HostDataSource struct {
	client client.Client
}

type hostDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Hostname types.String `tfsdk:"hostname"`
	UUID     types.String `tfsdk:"uuid"`
	Vendor   types.String `tfsdk:"vendor"`
	Model    types.String `tfsdk:"model"`
	Version  types.String `tfsdk:"version"`
}

func NewHostDataSource() datasource.DataSource { return &HostDataSource{} }

func (d *HostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *HostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads current host information from procurator.core Host service.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true},
			"name":     schema.StringAttribute{Computed: true},
			"hostname": schema.StringAttribute{Computed: true},
			"uuid":     schema.StringAttribute{Computed: true},
			"vendor":   schema.StringAttribute{Computed: true},
			"model":    schema.StringAttribute{Computed: true},
			"version":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *HostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData).client
}

func (d *HostDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	host, err := d.client.GetHost(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read host", err.Error())
		return
	}
	state := hostDataSourceModel{
		ID:       stringValue(host.ID),
		Name:     stringValue(host.Name),
		Hostname: stringValue(host.Hostname),
		UUID:     stringValue(host.UUID),
		Vendor:   stringValue(host.Vendor),
		Model:    stringValue(host.Model),
		Version:  stringValue(host.Version),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
