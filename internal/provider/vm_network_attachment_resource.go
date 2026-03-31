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

var _ resource.Resource = &VMNetworkAttachmentResource{}
var _ resource.ResourceWithConfigure = &VMNetworkAttachmentResource{}
var _ resource.ResourceWithImportState = &VMNetworkAttachmentResource{}

type VMNetworkAttachmentResource struct{ client client.Client }

type vmNetworkAttachmentModel struct {
	ID        types.String `tfsdk:"id"`
	VMID      types.String `tfsdk:"vm_id"`
	Network   types.String `tfsdk:"network"`
	NetBridge types.String `tfsdk:"net_bridge"`
	Target    types.String `tfsdk:"target"`
	Model     types.String `tfsdk:"model"`
	BootOrder types.Int64  `tfsdk:"boot_order"`
	VLAN      types.Int64  `tfsdk:"vlan"`
	MAC       types.String `tfsdk:"mac"`
}

func NewVMNetworkAttachmentResource() resource.Resource { return &VMNetworkAttachmentResource{} }
func (r *VMNetworkAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_network_attachment"
}
func (r *VMNetworkAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*providerData).client
	}
}
func (r *VMNetworkAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":         schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"vm_id":      schema.StringAttribute{Required: true},
		"network":    schema.StringAttribute{Optional: true},
		"net_bridge": schema.StringAttribute{Optional: true},
		"target":     schema.StringAttribute{Optional: true},
		"model":      schema.StringAttribute{Optional: true},
		"boot_order": schema.Int64Attribute{Optional: true},
		"vlan":       schema.Int64Attribute{Optional: true},
		"mac":        schema.StringAttribute{Computed: true},
	}}
}
func (r *VMNetworkAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmNetworkAttachmentModel
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
	nic := client.NetworkDevice{
		Network:   plan.Network.ValueString(),
		NetBridge: plan.NetBridge.ValueString(),
		Target:    plan.Target.ValueString(),
		Model:     plan.Model.ValueString(),
		VLAN:      int32(plan.VLAN.ValueInt64()),
		Attach:    true,
	}
	if !plan.BootOrder.IsNull() && !plan.BootOrder.IsUnknown() {
		nic.BootOrder = int32(plan.BootOrder.ValueInt64())
	}
	upd.NetworkDevices = append(upd.NetworkDevices, nic)
	taskID, err := r.client.UpdateVM(ctx, &upd)
	if err != nil {
		resp.Diagnostics.AddError("VM update failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after network attach", err.Error())
		return
	}
	updated, err := r.client.GetVM(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM after network attach", err.Error())
		return
	}
	actual, _ := findNICAttachment(updated.NetworkDevices, nic)
	if actual == nil {
		resp.Diagnostics.AddError("Attached network device not found", "NIC was not found on VM after update.")
		return
	}
	state := plan
	state.ID = types.StringValue(fmt.Sprintf("%s:%s", plan.VMID.ValueString(), firstNonEmpty(actual.Target, actual.MAC, actual.Network)))
	state.Target = stringValue(actual.Target)
	state.MAC = stringValue(actual.MAC)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *VMNetworkAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmNetworkAttachmentModel
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
	actual, _ := findNICByState(vm.NetworkDevices, state)
	if actual == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Target = stringValue(actual.Target)
	state.MAC = stringValue(actual.MAC)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *VMNetworkAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Recreate the network attachment resource to change parameters.")
}
func (r *VMNetworkAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmNetworkAttachmentModel
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
	actual, _ := findNICByState(vm.NetworkDevices, state)
	if actual == nil {
		return
	}
	upd := *vm
	filtered := make([]client.NetworkDevice, 0, len(upd.NetworkDevices))
	removed := false
	for i := range upd.NetworkDevices {
		n := upd.NetworkDevices[i]
		if !removed && ((actual.MAC != "" && n.MAC == actual.MAC) || (actual.Target != "" && n.Target == actual.Target)) {
			removed = true
			continue
		}
		filtered = append(filtered, n)
	}
	upd.NetworkDevices = filtered
	taskID, err := r.client.UpdateVM(ctx, &upd)
	if err != nil {
		resp.Diagnostics.AddError("VM update failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after network detach", err.Error())
		return
	}
}
func (r *VMNetworkAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected vm_id:nic_target_or_mac")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func findNICAttachment(nics []client.NetworkDevice, want client.NetworkDevice) (*client.NetworkDevice, int) {
	for i := range nics {
		n := &nics[i]
		if want.Target != "" && n.Target == want.Target {
			return n, i
		}
		if want.Network != "" && n.Network == want.Network && (want.BootOrder == 0 || n.BootOrder == want.BootOrder) {
			return n, i
		}
	}
	return nil, -1
}
func findNICByState(nics []client.NetworkDevice, st vmNetworkAttachmentModel) (*client.NetworkDevice, int) {
	for i := range nics {
		n := &nics[i]
		if !st.Target.IsNull() && st.Target.ValueString() != "" && n.Target == st.Target.ValueString() {
			return n, i
		}
		if !st.MAC.IsNull() && st.MAC.ValueString() != "" && n.MAC == st.MAC.ValueString() {
			return n, i
		}
		if !st.Network.IsNull() && st.Network.ValueString() != "" && n.Network == st.Network.ValueString() {
			return n, i
		}
	}
	return nil, -1
}
