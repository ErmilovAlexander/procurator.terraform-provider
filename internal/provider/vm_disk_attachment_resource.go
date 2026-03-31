package provider

import (
	"context"
	"fmt"
	"strings"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VMDiskAttachmentResource{}
var _ resource.ResourceWithConfigure = &VMDiskAttachmentResource{}
var _ resource.ResourceWithImportState = &VMDiskAttachmentResource{}

type VMDiskAttachmentResource struct{ client client.Client }

type vmDiskAttachmentModel struct {
	ID             types.String `tfsdk:"id"`
	VMID           types.String `tfsdk:"vm_id"`
	SizeGB         types.Int64  `tfsdk:"size_gb"`
	Source         types.String `tfsdk:"source"`
	StorageID      types.String `tfsdk:"storage_id"`
	DeviceType     types.String `tfsdk:"device_type"`
	Bus            types.String `tfsdk:"bus"`
	Target         types.String `tfsdk:"target"`
	BootOrder      types.Int64  `tfsdk:"boot_order"`
	ProvisionType  types.String `tfsdk:"provision_type"`
	DiskMode       types.String `tfsdk:"disk_mode"`
	ReadOnly       types.Bool   `tfsdk:"read_only"`
	RemoveOnDetach types.Bool   `tfsdk:"remove_on_detach"`
}

func NewVMDiskAttachmentResource() resource.Resource { return &VMDiskAttachmentResource{} }
func (r *VMDiskAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_disk_attachment"
}
func (r *VMDiskAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*providerData).client
	}
}
func (r *VMDiskAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"vm_id":            schema.StringAttribute{Required: true},
		"size_gb":          schema.Int64Attribute{Required: true},
		"source":           schema.StringAttribute{Optional: true},
		"storage_id":       schema.StringAttribute{Required: true},
		"device_type":      schema.StringAttribute{Optional: true},
		"bus":              schema.StringAttribute{Optional: true},
		"target":           schema.StringAttribute{Optional: true},
		"boot_order":       schema.Int64Attribute{Optional: true},
		"provision_type":   schema.StringAttribute{Optional: true},
		"disk_mode":        schema.StringAttribute{Optional: true},
		"read_only":        schema.BoolAttribute{Optional: true},
		"remove_on_detach": schema.BoolAttribute{Optional: true},
	}}
}
func (r *VMDiskAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmDiskAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.client.GetVM(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM", err.Error())
		return
	}
	upd := *vm
	disk := client.DiskDevice{Size: uint64(plan.SizeGB.ValueInt64()) * 1024, Source: plan.Source.ValueString(), StorageID: plan.StorageID.ValueString(), DeviceType: plan.DeviceType.ValueString(), Bus: plan.Bus.ValueString(), Target: plan.Target.ValueString(), BootOrder: int32(plan.BootOrder.ValueInt64()), ProvisionType: plan.ProvisionType.ValueString(), DiskMode: plan.DiskMode.ValueString(), ReadOnly: boolFrom(plan.ReadOnly), Attach: true, Create: plan.Source.ValueString() == ""}
	upd.DiskDevices = append(upd.DiskDevices, disk)
	taskID, err := r.client.UpdateVM(ctx, &upd)
	if err != nil {
		resp.Diagnostics.AddError("VM update failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after disk attach", err.Error())
		return
	}
	updated, err := r.client.GetVM(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM after disk attach", err.Error())
		return
	}
	actual, idx := findDiskAttachment(updated.DiskDevices, disk)
	if idx < 0 {
		resp.Diagnostics.AddError("Attached disk not found", "Disk was not found on VM after update.")
		return
	}
	state := plan
	state.ID = types.StringValue(fmt.Sprintf("%s:%s", plan.VMID.ValueString(), actual.Target))
	if state.Target.IsNull() || state.Target.IsUnknown() || state.Target.ValueString() == "" {
		state.Target = stringValue(actual.Target)
	}
	if plan.Source.IsNull() || plan.Source.IsUnknown() || plan.Source.ValueString() == "" {
		state.Source = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *VMDiskAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmDiskAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.client.GetVM(ctx, state.VMID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read VM", err.Error())
		return
	}
	actual, _ := findDiskByState(vm.DiskDevices, state)
	if actual == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if state.Target.IsNull() || state.Target.IsUnknown() || state.Target.ValueString() == "" {
		state.Target = stringValue(actual.Target)
	}
	if state.Source.IsNull() || state.Source.IsUnknown() || state.Source.ValueString() == "" {
		state.Source = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *VMDiskAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Recreate the disk attachment resource to change parameters.")
}
func (r *VMDiskAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmDiskAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.client.GetVM(ctx, state.VMID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError("Failed to read VM", err.Error())
		return
	}
	actual, _ := findDiskByState(vm.DiskDevices, state)
	if actual == nil {
		return
	}
	upd := *vm
	for i := range upd.DiskDevices {
		if upd.DiskDevices[i].Target == actual.Target {
			upd.DiskDevices[i].Detach = true
			upd.DiskDevices[i].Remove = boolFrom(state.RemoveOnDetach)
			break
		}
	}
	taskID, err := r.client.UpdateVM(ctx, &upd)
	if err != nil {
		resp.Diagnostics.AddError("VM update failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after disk detach", err.Error())
		return
	}
}
func (r *VMDiskAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected vm_id:disk_target")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func findDiskAttachment(disks []client.DiskDevice, want client.DiskDevice) (client.DiskDevice, int) {
	for i, d := range disks {
		if want.Target != "" && d.Target == want.Target {
			return d, i
		}
		if want.Source != "" && d.Source == want.Source {
			return d, i
		}
	}
	for i, d := range disks {
		if d.DeviceType == firstNonEmpty(want.DeviceType, "disk") && d.StorageID == want.StorageID {
			return d, i
		}
	}
	return client.DiskDevice{}, -1
}
func findDiskByState(disks []client.DiskDevice, st vmDiskAttachmentModel) (*client.DiskDevice, int) {
	for i := range disks {
		d := &disks[i]
		if !st.Target.IsNull() && st.Target.ValueString() != "" && d.Target == st.Target.ValueString() {
			return d, i
		}
		if !st.Source.IsNull() && st.Source.ValueString() != "" && d.Source == st.Source.ValueString() {
			return d, i
		}
	}
	return nil, -1
}
