package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &StorageDevicesDataSource{}
var _ datasource.DataSourceWithConfigure = &StorageDevicesDataSource{}

type StorageDevicesDataSource struct{ client client.StorageClient }

type storageDevicesDataSourceModel struct {
	Items []storageDeviceItemModel `tfsdk:"items"`
}

type storageDeviceItemModel struct {
	Adapter              stringValueType `tfsdk:"adapter"`
	CapacityMB           int64ValueType  `tfsdk:"capacity_mb"`
	DatastoreType        stringValueType `tfsdk:"datastore_type"`
	DatastoreID          stringValueType `tfsdk:"datastore_id"`
	DatastoreName        stringValueType `tfsdk:"datastore_name"`
	DriveType            stringValueType `tfsdk:"drive_type"`
	HardwareAcceleration stringValueType `tfsdk:"hardware_acceleration"`
	ID                   stringValueType `tfsdk:"id"`
	Identifier           stringValueType `tfsdk:"identifier"`
	LUN                  int64ValueType  `tfsdk:"lun"`
	Name                 stringValueType `tfsdk:"name"`
	OperationalState     stringValueType `tfsdk:"operational_state"`
	Owner                stringValueType `tfsdk:"owner"`
	PerenniallyReserved  stringValueType `tfsdk:"perennially_reserved"`
	PhysicalLocation     stringValueType `tfsdk:"physical_location"`
	SectorFormat         stringValueType `tfsdk:"sector_format"`
	Transport            stringValueType `tfsdk:"transport"`
	Type                 stringValueType `tfsdk:"type"`
	StorageInterface     stringValueType `tfsdk:"storage_interface"`
}

// aliases to keep struct declaration compact
type stringValueType = types.String
type int64ValueType = types.Int64

func NewStorageDevicesDataSource() datasource.DataSource { return &StorageDevicesDataSource{} }

func (d *StorageDevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_devices"
}

func (d *StorageDevicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*providerData)
	}
}

func (d *StorageDevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"items": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"adapter":               schema.StringAttribute{Computed: true},
				"capacity_mb":           schema.Int64Attribute{Computed: true},
				"datastore_type":        schema.StringAttribute{Computed: true},
				"datastore_id":          schema.StringAttribute{Computed: true},
				"datastore_name":        schema.StringAttribute{Computed: true},
				"drive_type":            schema.StringAttribute{Computed: true},
				"hardware_acceleration": schema.StringAttribute{Computed: true},
				"id":                    schema.StringAttribute{Computed: true},
				"identifier":            schema.StringAttribute{Computed: true},
				"lun":                   schema.Int64Attribute{Computed: true},
				"name":                  schema.StringAttribute{Computed: true},
				"operational_state":     schema.StringAttribute{Computed: true},
				"owner":                 schema.StringAttribute{Computed: true},
				"perennially_reserved":  schema.StringAttribute{Computed: true},
				"physical_location":     schema.StringAttribute{Computed: true},
				"sector_format":         schema.StringAttribute{Computed: true},
				"transport":             schema.StringAttribute{Computed: true},
				"type":                  schema.StringAttribute{Computed: true},
				"storage_interface":     schema.StringAttribute{Computed: true},
			}},
		},
	}}
}

func (d *StorageDevicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data storageDevicesDataSourceModel
	items, err := d.client.ListStorageDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list storage devices", err.Error())
		return
	}
	for _, it := range items {
		data.Items = append(data.Items, storageDeviceItemModel{
			Adapter:              stringValue(it.Adapter),
			CapacityMB:           int64Value(int64(it.CapacityMB)),
			DatastoreType:        stringValue(it.DatastoreType),
			DatastoreID:          stringValue(it.DatastoreID),
			DatastoreName:        stringValue(it.DatastoreName),
			DriveType:            stringValue(it.DriveType),
			HardwareAcceleration: stringValue(it.HardwareAcceleration),
			ID:                   stringValue(it.ID),
			Identifier:           stringValue(it.Identifier),
			LUN:                  int64Value(int64(it.LUN)),
			Name:                 stringValue(it.Name),
			OperationalState:     stringValue(it.OperationalState),
			Owner:                stringValue(it.Owner),
			PerenniallyReserved:  stringValue(it.PerenniallyReserved),
			PhysicalLocation:     stringValue(it.PhysicalLocation),
			SectorFormat:         stringValue(it.SectorFormat),
			Transport:            stringValue(it.Transport),
			Type:                 stringValue(it.Type),
			StorageInterface:     stringValue(it.StorageInterface),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
