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

var _ resource.Resource = &VMConvertToTemplateResource{}
var _ resource.ResourceWithConfigure = &VMConvertToTemplateResource{}
var _ resource.ResourceWithImportState = &VMConvertToTemplateResource{}

type VMConvertToTemplateResource struct{ client client.Client }

type vmConvertToTemplateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	VMID           types.String `tfsdk:"vm_id"`
	Name           types.String `tfsdk:"name"`
	UUID           types.String `tfsdk:"uuid"`
	Compatibility  types.String `tfsdk:"compatibility"`
	GuestOSFamily  types.String `tfsdk:"guest_os_family"`
	GuestOSVersion types.String `tfsdk:"guest_os_version"`
	MachineType    types.String `tfsdk:"machine_type"`
	StorageID      types.String `tfsdk:"storage_id"`
	StorageFolder  types.String `tfsdk:"storage_folder"`
	VCPUs          types.Int64  `tfsdk:"vcpus"`
	MemorySizeMB   types.Int64  `tfsdk:"memory_size_mb"`
	IsTemplate     types.Bool   `tfsdk:"is_template"`
}

func NewVMConvertToTemplateResource() resource.Resource { return &VMConvertToTemplateResource{} }

func (r *VMConvertToTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_convert_to_template"
}

func (r *VMConvertToTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}

func (r *VMConvertToTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Converts an existing powered-off VM to a template in-place using ConvertVmToTemplate2. Deleting this resource converts the template back to a VM using ConvertTemplateToVm2.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"vm_id":            schema.StringAttribute{Required: true},
			"name":             schema.StringAttribute{Computed: true},
			"uuid":             schema.StringAttribute{Computed: true},
			"compatibility":    schema.StringAttribute{Computed: true},
			"guest_os_family":  schema.StringAttribute{Computed: true},
			"guest_os_version": schema.StringAttribute{Computed: true},
			"machine_type":     schema.StringAttribute{Computed: true},
			"storage_id":       schema.StringAttribute{Computed: true},
			"storage_folder":   schema.StringAttribute{Computed: true},
			"vcpus":            schema.Int64Attribute{Computed: true},
			"memory_size_mb":   schema.Int64Attribute{Computed: true},
			"is_template":      schema.BoolAttribute{Computed: true},
		},
	}
}

func flattenVMConvertToTemplateResource(in vmConvertToTemplateResourceModel, tmpl client.VM) vmConvertToTemplateResourceModel {
	return vmConvertToTemplateResourceModel{
		ID:             stringValue(firstNonEmpty(tmpl.DeploymentName, tmpl.UUID, tmpl.Name, in.VMID.ValueString())),
		VMID:           stringValue(firstNonEmpty(in.VMID.ValueString(), tmpl.DeploymentName, tmpl.UUID, tmpl.Name)),
		Name:           stringValue(tmpl.Name),
		UUID:           stringValue(tmpl.UUID),
		Compatibility:  stringValue(tmpl.Compatibility),
		GuestOSFamily:  stringValue(tmpl.GuestOSFamily),
		GuestOSVersion: stringValue(tmpl.GuestOSVersion),
		MachineType:    stringValue(tmpl.MachineType),
		StorageID:      stringValue(tmpl.Storage.ID),
		StorageFolder:  stringValue(tmpl.Storage.Folder),
		VCPUs:          types.Int64Value(int64(tmpl.CPU.VCPUs)),
		MemorySizeMB:   types.Int64Value(int64(tmpl.Memory.SizeMB)),
		IsTemplate:     boolValue(tmpl.IsTemplate),
	}
}

func (r *VMConvertToTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmConvertToTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	taskID, err := r.client.ConvertVMToTemplate(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("VM convert to template failed", err.Error())
		return
	}
	if taskID != "" {
		task, err := r.client.WaitTask(ctx, taskID)
		if err != nil {
			resp.Diagnostics.AddError("Task wait failed after vm convert to template", err.Error())
			return
		}
		if task != nil && task.Status != 2 {
			resp.Diagnostics.AddError("Task wait failed after vm convert to template", firstNonEmpty(task.Error, "task ended in non-success status"))
			return
		}
	}

	tmpl, err := r.client.GetTemplate(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read converted template", err.Error())
		return
	}
	state := flattenVMConvertToTemplateResource(plan, *tmpl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMConvertToTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmConvertToTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tmpl, err := r.client.GetTemplate(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read converted template", err.Error())
		return
	}
	if !tmpl.IsTemplate {
		resp.State.RemoveResource(ctx)
		return
	}
	newState := flattenVMConvertToTemplateResource(state, *tmpl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VMConvertToTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmConvertToTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state vmConvertToTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.VMID.ValueString() != state.VMID.ValueString() {
		resp.Diagnostics.AddError("Unsupported update", "vm_id cannot be changed; recreate the resource.")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMConvertToTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmConvertToTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.ConvertTemplateToVM(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError("Template convert back to VM failed", err.Error())
		return
	}
	if taskID != "" {
		task, err := r.client.WaitTask(ctx, taskID)
		if err != nil {
			resp.Diagnostics.AddError("Task wait failed after template convert to VM", err.Error())
			return
		}
		if task != nil && task.Status != 2 {
			resp.Diagnostics.AddError("Task wait failed after template convert to VM", firstNonEmpty(task.Error, "task ended in non-success status"))
			return
		}
	}
}

func (r *VMConvertToTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
