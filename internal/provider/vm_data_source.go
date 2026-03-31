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
	ID             types.String `tfsdk:"id"`
	DeploymentName types.String `tfsdk:"deployment_name"`
	Name           types.String `tfsdk:"name"`
	UUID           types.String `tfsdk:"uuid"`
	Compatibility  types.String `tfsdk:"compatibility"`
	GuestOSFamily  types.String `tfsdk:"guest_os_family"`
	GuestOSVersion types.String `tfsdk:"guest_os_version"`
	MachineType    types.String `tfsdk:"machine_type"`
	StorageID      types.String `tfsdk:"storage_id"`
	StorageFolder  types.String `tfsdk:"storage_folder"`
	VCPUs          types.Int64  `tfsdk:"vcpus"`
	MemorySizeMB   types.Int64  `tfsdk:"memory_size_mb"`
	IsTemplate     types.Bool   `tfsdk:"is_template"`
}

func NewVMDataSource() datasource.DataSource { return &VMDataSource{} }

func (d *VMDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VMDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a VM by deployment_name, uuid or name using Vms.List.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Optional: true, Computed: true},
			"deployment_name":  schema.StringAttribute{Optional: true, Computed: true},
			"name":             schema.StringAttribute{Optional: true, Computed: true},
			"uuid":             schema.StringAttribute{Optional: true, Computed: true},
			"compatibility":    schema.StringAttribute{Computed: true},
			"guest_os_family":  schema.StringAttribute{Computed: true},
			"guest_os_version": schema.StringAttribute{Computed: true},
			"machine_type":     schema.StringAttribute{Computed: true},
			"storage_id":       schema.StringAttribute{Computed: true},
			"storage_folder":   schema.StringAttribute{Computed: true},
			"vcpus":            schema.Int64Attribute{Computed: true},
			"memory_size_mb":   schema.Int64Attribute{Computed: true},
			"is_template":      schema.BoolAttribute{Computed: true},
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

	state := vmDataSourceModel{
		ID:             stringValue(match.DeploymentName),
		DeploymentName: stringValue(match.DeploymentName),
		Name:           stringValue(match.Name),
		UUID:           stringValue(match.UUID),
		Compatibility:  stringValue(match.Compatibility),
		GuestOSFamily:  stringValue(match.GuestOSFamily),
		GuestOSVersion: stringValue(match.GuestOSVersion),
		MachineType:    stringValue(match.MachineType),
		StorageID:      stringValue(match.Storage.ID),
		StorageFolder:  stringValue(match.Storage.Folder),
		VCPUs:          types.Int64Value(int64(match.CPU.VCPUs)),
		MemorySizeMB:   types.Int64Value(int64(match.Memory.SizeMB)),
		IsTemplate:     boolValue(match.IsTemplate),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
