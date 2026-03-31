package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DatastoreResource{}
var _ resource.ResourceWithConfigure = &DatastoreResource{}
var _ resource.ResourceWithImportState = &DatastoreResource{}

type DatastoreResource struct{ client client.Client }

type datastoreResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	Name             types.String  `tfsdk:"name"`
	TypeCode         types.Int64   `tfsdk:"type_code"`
	Server           types.String  `tfsdk:"server"`
	Folder           types.String  `tfsdk:"folder"`
	Readonly         types.Bool    `tfsdk:"readonly"`
	Devices          types.List    `tfsdk:"devices"`
	Reinit           types.Bool    `tfsdk:"reinit"`
	NConnect         types.Int64   `tfsdk:"nconnect"`
	PoolName         types.String  `tfsdk:"pool_name"`
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

func NewDatastoreResource() resource.Resource { return &DatastoreResource{} }

func (r *DatastoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datastore"
}
func (r *DatastoreResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}
func (r *DatastoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":              schema.StringAttribute{Required: true},
		"type_code":         schema.Int64Attribute{Required: true},
		"server":            schema.StringAttribute{Optional: true},
		"folder":            schema.StringAttribute{Optional: true},
		"readonly":          schema.BoolAttribute{Optional: true},
		"devices":           schema.ListAttribute{Optional: true, ElementType: types.StringType},
		"reinit":            schema.BoolAttribute{Optional: true},
		"nconnect":          schema.Int64Attribute{Optional: true},
		"pool_name":         schema.StringAttribute{Computed: true},
		"state":             schema.Int64Attribute{Computed: true},
		"status":            schema.Int64Attribute{Computed: true},
		"drive_type":        schema.StringAttribute{Computed: true},
		"capacity_mb":       schema.Float64Attribute{Computed: true},
		"provisioned_mb":    schema.Float64Attribute{Computed: true},
		"free_mb":           schema.Float64Attribute{Computed: true},
		"used_mb":           schema.Float64Attribute{Computed: true},
		"thin_provisioning": schema.BoolAttribute{Computed: true},
		"access_mode":       schema.StringAttribute{Computed: true},
	}}
}

func (r *DatastoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datastoreResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ds := &client.Datastore{Name: plan.Name.ValueString(), TypeCode: int32(plan.TypeCode.ValueInt64()), Server: plan.Server.ValueString(), Folder: plan.Folder.ValueString(), Readonly: boolFrom(plan.Readonly)}
	if !plan.Devices.IsNull() && !plan.Devices.IsUnknown() {
		var devices []string
		resp.Diagnostics.Append(plan.Devices.ElementsAs(ctx, &devices, false)...)
		ds.Devices = devices
	}
	if !plan.Reinit.IsNull() && !plan.Reinit.IsUnknown() {
		b := plan.Reinit.ValueBool()
		ds.Reinit = &b
	}
	if !plan.NConnect.IsNull() && !plan.NConnect.IsUnknown() {
		n := int32(plan.NConnect.ValueInt64())
		ds.NConnect = &n
	}
	taskID, err := r.client.CreateDatastore(ctx, ds)
	if err != nil {
		resp.Diagnostics.AddError("Datastore create failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore create", err.Error())
		return
	}
	items, err := r.client.ListDatastores(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list datastores", err.Error())
		return
	}
	match := selectDatastore(items, "", plan.Name.ValueString())
	if match == nil {
		resp.Diagnostics.AddError("Datastore not found", "Created datastore was not found by name")
		return
	}
	state := flattenDatastoreResource(plan, *match)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *DatastoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datastoreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ds, err := r.client.GetDatastore(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read datastore", err.Error())
		return
	}
	newState := flattenDatastoreResource(state, *ds)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
func (r *DatastoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported update", "Datastore update is not supported; recreate the datastore resource.")
}
func (r *DatastoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datastoreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteDatastore(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Datastore delete failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore delete", err.Error())
		return
	}
}
func (r *DatastoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenDatastoreResource(base datastoreResourceModel, ds client.Datastore) datastoreResourceModel {
	out := base
	out.ID = stringValue(ds.ID)
	if out.Name.IsNull() || out.Name.IsUnknown() || out.Name.ValueString() == "" {
		out.Name = stringValue(ds.Name)
	}
	if out.TypeCode.IsNull() || out.TypeCode.IsUnknown() || out.TypeCode.ValueInt64() == 0 {
		out.TypeCode = types.Int64Value(int64(ds.TypeCode))
	}
	if out.Server.IsNull() || out.Server.IsUnknown() || out.Server.ValueString() == "" {
		out.Server = stringValue(ds.Server)
	}
	out.PoolName = stringValue(ds.PoolName)
	out.State = types.Int64Value(int64(ds.State))
	out.Status = types.Int64Value(int64(ds.Status))
	out.DriveType = stringValue(ds.DriveType)
	out.CapacityMB = types.Float64Value(ds.CapacityMB)
	out.ProvisionedMB = types.Float64Value(ds.ProvisionedMB)
	out.FreeMB = types.Float64Value(ds.FreeMB)
	out.UsedMB = types.Float64Value(ds.UsedMB)
	out.ThinProvisioning = boolValue(ds.ThinProvisioning)
	out.AccessMode = stringValue(ds.AccessMode)
	return out
}
