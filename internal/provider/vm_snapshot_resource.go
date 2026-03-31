package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VMSnapshotResource{}
var _ resource.ResourceWithConfigure = &VMSnapshotResource{}
var _ resource.ResourceWithImportState = &VMSnapshotResource{}

type VMSnapshotResource struct{ client client.Client }

type vmSnapshotDiskModel struct {
	Source types.String `tfsdk:"source"`
	Target types.String `tfsdk:"target"`
}

type vmSnapshotResourceModel struct {
	ID            types.String          `tfsdk:"id"`
	VMID          types.String          `tfsdk:"vm_id"`
	SnapshotID    types.Int64           `tfsdk:"snapshot_id"`
	Name          types.String          `tfsdk:"name"`
	Description   types.String          `tfsdk:"description"`
	IncludeMemory types.Bool            `tfsdk:"include_memory"`
	QuiesceFS     types.Bool            `tfsdk:"quiesce_fs"`
	Timestamp     types.Int64           `tfsdk:"timestamp"`
	Size          types.Int64           `tfsdk:"size"`
	Current       types.Bool            `tfsdk:"current"`
	MemorySource  types.String          `tfsdk:"memory_source"`
	ParentID      types.Int64           `tfsdk:"parent_id"`
	Disks         []vmSnapshotDiskModel `tfsdk:"disks"`
}

func NewVMSnapshotResource() resource.Resource { return &VMSnapshotResource{} }

func (r *VMSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_snapshot"
}
func (r *VMSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}
func (r *VMSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"vm_id":          schema.StringAttribute{Required: true},
		"snapshot_id":    schema.Int64Attribute{Computed: true},
		"name":           schema.StringAttribute{Required: true},
		"description":    schema.StringAttribute{Optional: true},
		"include_memory": schema.BoolAttribute{Optional: true},
		"quiesce_fs":     schema.BoolAttribute{Optional: true},
		"timestamp":      schema.Int64Attribute{Computed: true},
		"size":           schema.Int64Attribute{Computed: true},
		"current":        schema.BoolAttribute{Computed: true},
		"memory_source":  schema.StringAttribute{Computed: true},
		"parent_id":      schema.Int64Attribute{Computed: true},
	}, Blocks: map[string]schema.Block{
		"disks": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"source": schema.StringAttribute{Computed: true},
			"target": schema.StringAttribute{Computed: true},
		}}},
	}}
}

func (r *VMSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmSnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.TakeVMSnapshot(ctx, plan.VMID.ValueString(), plan.Name.ValueString(), plan.Description.ValueString(), boolFrom(plan.IncludeMemory), boolFrom(plan.QuiesceFS))
	if err != nil {
		resp.Diagnostics.AddError("Snapshot create failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after snapshot create", err.Error())
		return
	}
	state, err := r.readSnapshotState(ctx, plan.VMID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created snapshot", err.Error())
		return
	}
	state.Description = plan.Description
	state.IncludeMemory = plan.IncludeMemory
	state.QuiesceFS = plan.QuiesceFS
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmSnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	snaps, currentID, err := r.client.ListVMSnapshots(ctx, state.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM snapshots", err.Error())
		return
	}
	for _, s := range snaps {
		if s.ID == state.SnapshotID.ValueInt64() {
			newState := flattenSnapshot(state.VMID.ValueString(), s, currentID)
			newState.Description = state.Description
			newState.IncludeMemory = state.IncludeMemory
			newState.QuiesceFS = state.QuiesceFS
			resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *VMSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Snapshot update not supported", "Recreate the resource if you need a new snapshot.")
}

func (r *VMSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmSnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteVMSnapshot(ctx, state.VMID.ValueString(), state.SnapshotID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Snapshot delete failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after snapshot delete", err.Error())
		return
	}
}

func (r *VMSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected vm_id:snapshot_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_id"), parts[0])...)
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid snapshot_id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("snapshot_id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *VMSnapshotResource) readSnapshotState(ctx context.Context, vmID, name string) (vmSnapshotResourceModel, error) {
	snaps, currentID, err := r.client.ListVMSnapshots(ctx, vmID)
	if err != nil {
		return vmSnapshotResourceModel{}, err
	}
	for _, s := range snaps {
		if s.Name == name {
			return flattenSnapshot(vmID, s, currentID), nil
		}
	}
	return vmSnapshotResourceModel{}, fmt.Errorf("snapshot %q not found", name)
}

func flattenSnapshot(vmID string, s client.Snapshot, currentID int64) vmSnapshotResourceModel {
	out := vmSnapshotResourceModel{
		ID:            types.StringValue(fmt.Sprintf("%s:%d", vmID, s.ID)),
		VMID:          types.StringValue(vmID),
		SnapshotID:    types.Int64Value(s.ID),
		Name:          stringValue(s.Name),
		Description:   stringValue(s.Description),
		IncludeMemory: boolValue(s.MemoryEnabled),
		QuiesceFS:     boolValue(s.QuiesceFS),
		Timestamp:     types.Int64Value(s.Timestamp),
		Size:          types.Int64Value(s.Size),
		Current:       boolValue(s.ID == currentID),
		MemorySource:  stringValue(s.MemorySource),
		ParentID:      types.Int64Value(int64(s.ParentID)),
	}
	return out
}
