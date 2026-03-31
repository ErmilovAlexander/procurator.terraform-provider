package provider

import (
	"context"
	"time"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &NetworkResource{}
var _ resource.ResourceWithConfigure = &NetworkResource{}
var _ resource.ResourceWithImportState = &NetworkResource{}

type NetworkResource struct{ client client.UmbraClient }

type networkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	VLAN        types.Int64  `tfsdk:"vlan"`
	SwitchID    types.String `tfsdk:"switch_id"`
	NetBridge   types.String `tfsdk:"net_bridge"`
	Kind        types.String `tfsdk:"kind"`
	VmsCount    types.Int64  `tfsdk:"vms_count"`
	ActivePorts types.Int64  `tfsdk:"active_ports"`
	State       types.String `tfsdk:"state"`
	Errors      types.List   `tfsdk:"errors"`
}

func NewNetworkResource() resource.Resource { return &NetworkResource{} }

func (r *NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData)
}

func (r *NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":         schema.StringAttribute{Required: true},
		"vlan":         schema.Int64Attribute{Required: true},
		"switch_id":    schema.StringAttribute{Required: true},
		"net_bridge":   schema.StringAttribute{Computed: true},
		"kind":         schema.StringAttribute{Computed: true},
		"vms_count":    schema.Int64Attribute{Computed: true},
		"active_ports": schema.Int64Attribute{Computed: true},
		"state":        schema.StringAttribute{Computed: true},
		"errors":       schema.ListAttribute{Computed: true, ElementType: types.StringType},
	}}
}

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.CreateNetwork(ctx, &client.NetworkCreateRequest{
		Name:     plan.Name.ValueString(),
		VLAN:     uint32(plan.VLAN.ValueInt64()),
		SwitchID: plan.SwitchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Network create failed", err.Error())
		return
	}
	var net *client.Network
	for i := 0; i < 10; i++ {
		net, err = r.client.GetNetworkByName(ctx, plan.Name.ValueString())
		if err == nil {
			break
		}
		if err != client.ErrNotFound {
			resp.Diagnostics.AddError("Failed to read created network", err.Error())
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created network", err.Error())
		return
	}
	state := flattenNetworkResource(ctx, plan, *net)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	net, err := r.client.GetNetworkByID(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read network", err.Error())
		return
	}
	newState := flattenNetworkResource(ctx, state, *net)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.UpdateNetwork(ctx, &client.Network{
		ID:       plan.ID.ValueString(),
		Name:     plan.Name.ValueString(),
		VLAN:     int32(plan.VLAN.ValueInt64()),
		SwitchID: plan.SwitchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Network update failed", err.Error())
		return
	}
	net, err := r.client.GetNetworkByID(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated network", err.Error())
		return
	}
	state := flattenNetworkResource(ctx, plan, *net)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNetwork(ctx, state.ID.ValueString()); err != nil && err != client.ErrNotFound {
		resp.Diagnostics.AddError("Network delete failed", err.Error())
	}
}

func (r *NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenNetworkResource(ctx context.Context, base networkResourceModel, in client.Network) networkResourceModel {
	out := base
	out.ID = stringValue(in.ID)
	out.Name = stringValue(in.Name)
	out.VLAN = types.Int64Value(int64(in.VLAN))
	out.SwitchID = stringValue(in.SwitchID)
	out.NetBridge = stringValue(in.NetBridge)
	out.Kind = stringValue(in.Kind)
	out.VmsCount = types.Int64Value(int64(in.VmsCount))
	out.ActivePorts = types.Int64Value(int64(in.ActivePorts))
	out.State = stringValue(in.State)
	out.Errors = stringList(ctx, in.Errors)
	return out
}
