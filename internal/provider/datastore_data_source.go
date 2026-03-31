package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DatastoreDataSource{}
var _ datasource.DataSourceWithConfigure = &DatastoreDataSource{}

type DatastoreDataSource struct {
	client client.Client
}

type datastoreDataSourceModel struct {
	ID               types.String  `tfsdk:"id"`
	Name             types.String  `tfsdk:"name"`
	PoolName         types.String  `tfsdk:"pool_name"`
	TypeCode         types.Int64   `tfsdk:"type_code"`
	State            types.Int64   `tfsdk:"state"`
	Status           types.Int64   `tfsdk:"status"`
	DriveType        types.String  `tfsdk:"drive_type"`
	CapacityMB       types.Float64 `tfsdk:"capacity_mb"`
	ProvisionedMB    types.Float64 `tfsdk:"provisioned_mb"`
	FreeMB           types.Float64 `tfsdk:"free_mb"`
	UsedMB           types.Float64 `tfsdk:"used_mb"`
	ThinProvisioning types.Bool    `tfsdk:"thin_provisioning"`
	AccessMode       types.String  `tfsdk:"access_mode"`
}

func NewDatastoreDataSource() datasource.DataSource { return &DatastoreDataSource{} }

func (d *DatastoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datastore"
}

func (d *DatastoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a datastore by id or name via Datastores.List/Get.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Optional: true, Computed: true},
			"name":              schema.StringAttribute{Optional: true, Computed: true},
			"pool_name":         schema.StringAttribute{Computed: true},
			"type_code":         schema.Int64Attribute{Computed: true},
			"state":             schema.Int64Attribute{Computed: true},
			"status":            schema.Int64Attribute{Computed: true},
			"drive_type":        schema.StringAttribute{Computed: true},
			"capacity_mb":       schema.Float64Attribute{Computed: true},
			"provisioned_mb":    schema.Float64Attribute{Computed: true},
			"free_mb":           schema.Float64Attribute{Computed: true},
			"used_mb":           schema.Float64Attribute{Computed: true},
			"thin_provisioning": schema.BoolAttribute{Computed: true},
			"access_mode":       schema.StringAttribute{Computed: true},
		},
	}
}

func (d *DatastoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerData).client
}

func (d *DatastoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config datastoreDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListDatastores(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list datastores", err.Error())
		return
	}

	match := selectDatastore(items, config.ID.ValueString(), config.Name.ValueString())
	if match == nil {
		resp.Diagnostics.AddError("Datastore not found", "No datastore matched the provided id or name")
		return
	}

	state := datastoreDataSourceModel{
		ID:               stringValue(match.ID),
		Name:             stringValue(match.Name),
		PoolName:         stringValue(match.PoolName),
		TypeCode:         types.Int64Value(int64(match.TypeCode)),
		State:            types.Int64Value(int64(match.State)),
		Status:           types.Int64Value(int64(match.Status)),
		DriveType:        stringValue(match.DriveType),
		CapacityMB:       types.Float64Value(match.CapacityMB),
		ProvisionedMB:    types.Float64Value(match.ProvisionedMB),
		FreeMB:           types.Float64Value(match.FreeMB),
		UsedMB:           types.Float64Value(match.UsedMB),
		ThinProvisioning: boolValue(match.ThinProvisioning),
		AccessMode:       stringValue(match.AccessMode),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
