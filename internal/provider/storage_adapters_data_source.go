package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &StorageAdaptersDataSource{}
var _ datasource.DataSourceWithConfigure = &StorageAdaptersDataSource{}

type StorageAdaptersDataSource struct{ client client.StorageClient }

type storageAdaptersDataSourceModel struct {
	Items []storageAdapterItemModel `tfsdk:"items"`
}

type storageAdapterItemModel struct {
	Adapter              types.String `tfsdk:"adapter"`
	Devices              types.Int64  `tfsdk:"devices"`
	ID                   types.String `tfsdk:"id"`
	Identifier           types.String `tfsdk:"identifier"`
	Model                types.String `tfsdk:"model"`
	Path                 types.Int64  `tfsdk:"path"`
	StatusText           types.String `tfsdk:"status_text"`
	StatusValue          types.String `tfsdk:"status_value"`
	Targets              types.Int64  `tfsdk:"targets"`
	Type                 types.String `tfsdk:"type"`
	RescanStorageAdapter types.Bool   `tfsdk:"rescan_storage_adapter"`
	ScanStorageDevice    types.Bool   `tfsdk:"scan_storage_device"`
	ScanDatastore        types.Bool   `tfsdk:"scan_datastore"`
	ProtectDatastore     types.Bool   `tfsdk:"protect_datastore"`
	Response             types.String `tfsdk:"response"`
}

func NewStorageAdaptersDataSource() datasource.DataSource { return &StorageAdaptersDataSource{} }

func (d *StorageAdaptersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_adapters"
}

func (d *StorageAdaptersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*providerData)
	}
}

func (d *StorageAdaptersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"items": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"adapter":                schema.StringAttribute{Computed: true},
				"devices":                schema.Int64Attribute{Computed: true},
				"id":                     schema.StringAttribute{Computed: true},
				"identifier":             schema.StringAttribute{Computed: true},
				"model":                  schema.StringAttribute{Computed: true},
				"path":                   schema.Int64Attribute{Computed: true},
				"status_text":            schema.StringAttribute{Computed: true},
				"status_value":           schema.StringAttribute{Computed: true},
				"targets":                schema.Int64Attribute{Computed: true},
				"type":                   schema.StringAttribute{Computed: true},
				"rescan_storage_adapter": schema.BoolAttribute{Computed: true},
				"scan_storage_device":    schema.BoolAttribute{Computed: true},
				"scan_datastore":         schema.BoolAttribute{Computed: true},
				"protect_datastore":      schema.BoolAttribute{Computed: true},
				"response":               schema.StringAttribute{Computed: true},
			}},
		},
	}}
}

func (d *StorageAdaptersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data storageAdaptersDataSourceModel
	items, err := d.client.ListStorageAdapters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list storage adapters", err.Error())
		return
	}
	for _, it := range items {
		data.Items = append(data.Items, storageAdapterItemModel{
			Adapter:              stringValue(it.Adapter),
			Devices:              int64Value(int64(it.Devices)),
			ID:                   stringValue(it.ID),
			Identifier:           stringValue(it.Identifier),
			Model:                stringValue(it.Model),
			Path:                 int64Value(int64(it.Path)),
			StatusText:           stringValue(it.StatusText),
			StatusValue:          stringValue(it.StatusValue),
			Targets:              int64Value(int64(it.Targets)),
			Type:                 stringValue(it.Type),
			RescanStorageAdapter: boolValue(it.RescanStorageAdapter),
			ScanStorageDevice:    boolValue(it.ScanStorageDevice),
			ScanDatastore:        boolValue(it.ScanDatastore),
			ProtectDatastore:     boolValue(it.ProtectDatastore),
			Response:             stringValue(it.Response),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
