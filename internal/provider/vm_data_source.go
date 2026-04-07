package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VMDataSource{}
var _ datasource.DataSourceWithConfigure = &VMDataSource{}

type VMDataSource struct {
	client client.Client
}

type vmDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	DeploymentName    types.String `tfsdk:"deployment_name"`
	Name              types.String `tfsdk:"name"`
	UUID              types.String `tfsdk:"uuid"`
	Compatibility     types.String `tfsdk:"compatibility"`
	GuestOSFamily     types.String `tfsdk:"guest_os_family"`
	GuestOSVersion    types.String `tfsdk:"guest_os_version"`
	MachineType       types.String `tfsdk:"machine_type"`
	StorageID         types.String `tfsdk:"storage_id"`
	StorageFolder     types.String `tfsdk:"storage_folder"`
	VCPUs             types.Int64  `tfsdk:"vcpus"`
	MemorySizeMB      types.Int64  `tfsdk:"memory_size_mb"`
	IsTemplate        types.Bool   `tfsdk:"is_template"`
	GuestToolsStatus  types.String `tfsdk:"guest_tools_status"`
	GuestToolsVersion types.String `tfsdk:"guest_tools_version"`
	GuestIP           types.String `tfsdk:"guest_ip"`
	GuestDNSName      types.String `tfsdk:"guest_dns_name"`
}

func NewVMDataSource() datasource.DataSource { return &VMDataSource{} }

func (d *VMDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VMDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a VM by deployment_name, uuid or name using Vms.List and enriches runtime guest tools data using Vms.GetWithParams.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Optional: true, Computed: true},
			"deployment_name":     schema.StringAttribute{Optional: true, Computed: true},
			"name":                schema.StringAttribute{Optional: true, Computed: true},
			"uuid":                schema.StringAttribute{Optional: true, Computed: true},
			"compatibility":       schema.StringAttribute{Computed: true},
			"guest_os_family":     schema.StringAttribute{Computed: true},
			"guest_os_version":    schema.StringAttribute{Computed: true},
			"machine_type":        schema.StringAttribute{Computed: true},
			"storage_id":          schema.StringAttribute{Computed: true},
			"storage_folder":      schema.StringAttribute{Computed: true},
			"vcpus":               schema.Int64Attribute{Computed: true},
			"memory_size_mb":      schema.Int64Attribute{Computed: true},
			"is_template":         schema.BoolAttribute{Computed: true},
			"guest_tools_status":  schema.StringAttribute{Computed: true},
			"guest_tools_version": schema.StringAttribute{Computed: true},
			"guest_ip":            schema.StringAttribute{Computed: true},
			"guest_dns_name":      schema.StringAttribute{Computed: true},
		},
	}
}

func (d *VMDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData).client
}

func (d *VMDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config vmDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListVMs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list VMs", err.Error())
		return
	}

	match := selectVM(items, config.ID.ValueString(), config.DeploymentName.ValueString(), config.Name.ValueString(), config.UUID.ValueString())
	if match == nil {
		resp.Diagnostics.AddError("VM not found", "No VM matched deployment_name, uuid or name")
		return
	}

	full, err := d.client.GetVMWithParams(ctx, match.DeploymentName, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM with guest tools", err.Error())
		return
	}

	state := vmDataSourceModel{
		ID:                stringValue(full.DeploymentName),
		DeploymentName:    stringValue(full.DeploymentName),
		Name:              stringValue(full.Name),
		UUID:              stringValue(full.UUID),
		Compatibility:     stringValue(full.Compatibility),
		GuestOSFamily:     stringValue(full.GuestOSFamily),
		GuestOSVersion:    stringValue(full.GuestOSVersion),
		MachineType:       stringValue(full.MachineType),
		StorageID:         stringValue(full.Storage.ID),
		StorageFolder:     stringValue(full.Storage.Folder),
		VCPUs:             types.Int64Value(int64(full.CPU.VCPUs)),
		MemorySizeMB:      types.Int64Value(int64(full.Memory.SizeMB)),
		IsTemplate:        boolValue(full.IsTemplate),
		GuestToolsStatus:  stringValue(full.GuestToolsInfo.Status),
		GuestToolsVersion: stringValue(full.GuestToolsInfo.Version),
		GuestIP:           stringValue(full.GuestToolsInfo.IP),
		GuestDNSName:      stringValue(full.GuestToolsInfo.DNSName),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
