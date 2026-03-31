package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DatastoreLVMResource{}
var _ resource.ResourceWithConfigure = &DatastoreLVMResource{}
var _ resource.ResourceWithImportState = &DatastoreLVMResource{}

type DatastoreLVMResource struct{ client client.Client }

type datastoreLVMResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	Name             types.String  `tfsdk:"name"`
	Devices          types.List    `tfsdk:"devices"`
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

func NewDatastoreLVMResource() resource.Resource { return &DatastoreLVMResource{} }

func (r *DatastoreLVMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datastore_lvm"
}
func (r *DatastoreLVMResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}
func (r *DatastoreLVMResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":              schema.StringAttribute{Required: true},
		"devices":           schema.ListAttribute{Required: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
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

func (r *DatastoreLVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datastoreLVMResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var devices []string
	resp.Diagnostics.Append(plan.Devices.ElementsAs(ctx, &devices, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.CreateDatastore(ctx, &client.Datastore{Name: plan.Name.ValueString(), TypeCode: 2, Devices: devices})
	if err != nil {
		resp.Diagnostics.AddError("Datastore LVM create failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore LVM create", err.Error())
		return
	}
	items, err := r.client.ListDatastores(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list datastores", err.Error())
		return
	}
	match := selectDatastore(items, "", plan.Name.ValueString())
	if match == nil {
		resp.Diagnostics.AddError("Datastore not found", "Created LVM datastore was not found by name")
		return
	}
	state := flattenDatastoreLVMResource(plan, *match)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *DatastoreLVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datastoreLVMResourceModel
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
		resp.Diagnostics.AddError("Failed to read datastore LVM", err.Error())
		return
	}
	newState := flattenDatastoreLVMResource(state, *ds)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
func (r *DatastoreLVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported update", "Datastore LVM update is not supported; recreate the resource.")
}
func (r *DatastoreLVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datastoreLVMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteDatastore(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Datastore LVM delete failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore LVM delete", err.Error())
		return
	}
}
func (r *DatastoreLVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenDatastoreLVMResource(base datastoreLVMResourceModel, ds client.Datastore) datastoreLVMResourceModel {
	out := base
	out.ID = stringValue(ds.ID)
	if out.Name.IsNull() || out.Name.IsUnknown() || out.Name.ValueString() == "" {
		out.Name = stringValue(ds.Name)
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
