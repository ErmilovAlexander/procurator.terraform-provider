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

var _ resource.Resource = &DatastoreFolderResource{}
var _ resource.ResourceWithConfigure = &DatastoreFolderResource{}
var _ resource.ResourceWithImportState = &DatastoreFolderResource{}

type DatastoreFolderResource struct{ client client.Client }

type datastoreFolderResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Path types.String `tfsdk:"path"`
}

func NewDatastoreFolderResource() resource.Resource { return &DatastoreFolderResource{} }

func (r *DatastoreFolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datastore_folder"
}
func (r *DatastoreFolderResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}
func (r *DatastoreFolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"path": schema.StringAttribute{Required: true, Description: "Full folder path in format <storage_id>:/path/to/folder"},
	}}
}
func (r *DatastoreFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datastoreFolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.CreateDatastoreFolder(ctx, plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Datastore folder create failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore folder create", err.Error())
		return
	}
	_, err = r.client.BrowseDatastoreItem(ctx, plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created datastore folder", err.Error())
		return
	}
	plan.ID = plan.Path
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *DatastoreFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datastoreFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.BrowseDatastoreItem(ctx, state.Path.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read datastore folder", err.Error())
		return
	}
	state.ID = state.Path
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *DatastoreFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported update", "Datastore folder update is not supported; recreate the resource.")
}
func (r *DatastoreFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datastoreFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteDatastoreItem(ctx, []string{state.Path.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Datastore folder delete failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after datastore folder delete", err.Error())
		return
	}
}
func (r *DatastoreFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
