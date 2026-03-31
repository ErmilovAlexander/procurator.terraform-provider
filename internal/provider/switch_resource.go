package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SwitchResource{}
var _ resource.ResourceWithConfigure = &SwitchResource{}
var _ resource.ResourceWithImportState = &SwitchResource{}

type SwitchResource struct{ client client.UmbraClient }

type switchNICsModel struct {
	Active    types.List `tfsdk:"active"`
	Standby   types.List `tfsdk:"standby"`
	Unused    types.List `tfsdk:"unused"`
	Connected types.List `tfsdk:"connected"`
	Inherit   types.Bool `tfsdk:"inherit"`
}

type switchResourceModel struct {
	ID       types.String     `tfsdk:"id"`
	MTU      types.Int64      `tfsdk:"mtu"`
	NICs     *switchNICsModel `tfsdk:"nics"`
	Networks types.List       `tfsdk:"networks"`
	State    types.String     `tfsdk:"state"`
	Errors   types.List       `tfsdk:"errors"`
}

func NewSwitchResource() resource.Resource { return &SwitchResource{} }

func (r *SwitchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_switch"
}

func (r *SwitchResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData)
}

func (r *SwitchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mtu":      schema.Int64Attribute{Required: true},
			"networks": schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"state":    schema.StringAttribute{Computed: true},
			"errors":   schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"nics": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"active":    schema.ListAttribute{Optional: true, ElementType: types.StringType},
					"standby":   schema.ListAttribute{Optional: true, ElementType: types.StringType},
					"unused":    schema.ListAttribute{Optional: true, ElementType: types.StringType},
					"connected": schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"inherit":   schema.BoolAttribute{Optional: true},
				},
			},
		},
	}
}

func (r *SwitchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan switchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateSwitch(ctx, &client.SwitchCreateRequest{MTU: uint32(plan.MTU.ValueInt64()), NICs: expandSwitchNICs(ctx, plan.NICs, &resp.Diagnostics)})
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Switch create failed", err.Error())
		return
	}
	sw, err := r.client.GetSwitch(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created switch", err.Error())
		return
	}
	state := flattenSwitchResource(ctx, plan, *sw)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state switchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sw, err := r.client.GetSwitch(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read switch", err.Error())
		return
	}
	newState := flattenSwitchResource(ctx, state, *sw)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *SwitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan switchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.UpdateSwitch(ctx, &client.Switch{ID: plan.ID.ValueString(), MTU: uint32(plan.MTU.ValueInt64()), NICs: expandSwitchNICs(ctx, plan.NICs, &resp.Diagnostics)})
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Switch update failed", err.Error())
		return
	}
	sw, err := r.client.GetSwitch(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated switch", err.Error())
		return
	}
	state := flattenSwitchResource(ctx, plan, *sw)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwitchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state switchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSwitch(ctx, state.ID.ValueString()); err != nil && err != client.ErrNotFound {
		resp.Diagnostics.AddError("Switch delete failed", err.Error())
	}
}

func (r *SwitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func expandSwitchNICs(ctx context.Context, in *switchNICsModel, diags *diag.Diagnostics) client.NICs {
	if in == nil {
		return client.NICs{}
	}
	out := client.NICs{Inherit: boolFrom(in.Inherit)}
	if !in.Active.IsNull() && !in.Active.IsUnknown() {
		diags.Append(in.Active.ElementsAs(ctx, &out.Active, false)...)
	}
	if !in.Standby.IsNull() && !in.Standby.IsUnknown() {
		diags.Append(in.Standby.ElementsAs(ctx, &out.Standby, false)...)
	}
	if !in.Unused.IsNull() && !in.Unused.IsUnknown() {
		diags.Append(in.Unused.ElementsAs(ctx, &out.Unused, false)...)
	}
	if !in.Connected.IsNull() && !in.Connected.IsUnknown() {
		diags.Append(in.Connected.ElementsAs(ctx, &out.Connected, false)...)
	}
	return out
}

func flattenSwitchResource(ctx context.Context, base switchResourceModel, in client.Switch) switchResourceModel {
	out := base
	out.ID = stringValue(in.ID)
	out.MTU = types.Int64Value(int64(in.MTU))
	out.Networks = stringList(ctx, in.Networks)
	out.State = stringValue(in.State)
	out.Errors = stringList(ctx, in.Errors)

	// Если nics в плане не задавался вообще, и Umbra не вернула meaningful NIC config,
	// оставляем nics=nil, чтобы не получить "was null, but now object".
	if base.NICs == nil &&
		len(in.NICs.Active) == 0 &&
		len(in.NICs.Standby) == 0 &&
		len(in.NICs.Unused) == 0 &&
		len(in.NICs.Connected) == 0 &&
		!in.NICs.Inherit {
		out.NICs = nil
		return out
	}

	var active, standby, unused, connected types.List

	// active
	if len(in.NICs.Active) > 0 {
		active = stringList(ctx, in.NICs.Active)
	} else if base.NICs != nil && !base.NICs.Active.IsNull() && !base.NICs.Active.IsUnknown() {
		active = base.NICs.Active
	} else {
		active = types.ListNull(types.StringType)
	}

	// standby
	if len(in.NICs.Standby) > 0 {
		standby = stringList(ctx, in.NICs.Standby)
	} else if base.NICs != nil && !base.NICs.Standby.IsNull() && !base.NICs.Standby.IsUnknown() {
		standby = base.NICs.Standby
	} else {
		standby = types.ListNull(types.StringType)
	}

	// unused
	if len(in.NICs.Unused) > 0 {
		unused = stringList(ctx, in.NICs.Unused)
	} else if base.NICs != nil && !base.NICs.Unused.IsNull() && !base.NICs.Unused.IsUnknown() {
		unused = base.NICs.Unused
	} else {
		unused = types.ListNull(types.StringType)
	}

	// connected
	if len(in.NICs.Connected) > 0 {
		connected = stringList(ctx, in.NICs.Connected)
	} else if base.NICs != nil && !base.NICs.Connected.IsNull() && !base.NICs.Connected.IsUnknown() {
		connected = base.NICs.Connected
	} else {
		connected = types.ListNull(types.StringType)
	}

	out.NICs = &switchNICsModel{
		Active:    active,
		Standby:   standby,
		Unused:    unused,
		Connected: connected,
		Inherit:   boolValue(in.NICs.Inherit),
	}
	return out
}
