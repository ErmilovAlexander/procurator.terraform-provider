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

var _ resource.Resource = &TemplateResource{}
var _ resource.ResourceWithConfigure = &TemplateResource{}
var _ resource.ResourceWithImportState = &TemplateResource{}

type TemplateResource struct{ client client.Client }

type templateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SourceVMID     types.String `tfsdk:"source_vm_id"`
	StorageID      types.String `tfsdk:"storage_id"`
	Name           types.String `tfsdk:"name"`
	UUID           types.String `tfsdk:"uuid"`
	Compatibility  types.String `tfsdk:"compatibility"`
	GuestOSFamily  types.String `tfsdk:"guest_os_family"`
	GuestOSVersion types.String `tfsdk:"guest_os_version"`
	MachineType    types.String `tfsdk:"machine_type"`
	StorageFolder  types.String `tfsdk:"storage_folder"`
	VCPUs          types.Int64  `tfsdk:"vcpus"`
	MemorySizeMB   types.Int64  `tfsdk:"memory_size_mb"`
	IsTemplate     types.Bool   `tfsdk:"is_template"`
}

func NewTemplateResource() resource.Resource { return &TemplateResource{} }

func (r *TemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (r *TemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}

func (r *TemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a template from an existing VM using procurator.core template APIs.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"source_vm_id":     schema.StringAttribute{Required: true},
			"storage_id":       schema.StringAttribute{Required: true},
			"name":             schema.StringAttribute{Required: true},
			"uuid":             schema.StringAttribute{Computed: true},
			"compatibility":    schema.StringAttribute{Computed: true},
			"guest_os_family":  schema.StringAttribute{Computed: true},
			"guest_os_version": schema.StringAttribute{Computed: true},
			"machine_type":     schema.StringAttribute{Computed: true},
			"storage_folder":   schema.StringAttribute{Computed: true},
			"vcpus":            schema.Int64Attribute{Computed: true},
			"memory_size_mb":   schema.Int64Attribute{Computed: true},
			"is_template":      schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *TemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := r.client.CreateTemplateFromVM(ctx, plan.SourceVMID.ValueString(), plan.StorageID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Template create failed", err.Error())
		return
	}
	if id == "" {
		items, err := r.client.ListTemplates(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list templates after create", err.Error())
			return
		}
		match := selectVM(items, "", "", plan.Name.ValueString(), "")
		if match == nil {
			resp.Diagnostics.AddError("Template not found", "Created template was not found by name")
			return
		}
		id = firstNonEmpty(match.DeploymentName, match.UUID, match.Name)
	}
	tmpl, err := r.client.GetTemplate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created template", err.Error())
		return
	}
	state := flattenTemplateResource(plan, *tmpl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateResourceModel
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
		resp.Diagnostics.AddError("Failed to read template", err.Error())
		return
	}
	newState := flattenTemplateResource(state, *tmpl)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *TemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported update", "Template update is not supported; recreate the template resource.")
}

func (r *TemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteTemplate(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Template delete failed", err.Error())
		return
	}
	if taskID != "" {
		if _, err := r.client.WaitTask(ctx, taskID); err != nil {
			resp.Diagnostics.AddError("Task wait failed after template delete", err.Error())
			return
		}
	}
}

func (r *TemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenTemplateResource(base templateResourceModel, in client.VM) templateResourceModel {
	out := base
	out.ID = stringValue(firstNonEmpty(in.DeploymentName, in.UUID, in.Name))
	out.Name = stringValue(in.Name)
	out.UUID = stringValue(in.UUID)
	out.Compatibility = stringValue(in.Compatibility)
	out.GuestOSFamily = stringValue(in.GuestOSFamily)
	out.GuestOSVersion = stringValue(in.GuestOSVersion)
	out.MachineType = stringValue(in.MachineType)
	out.StorageFolder = stringValue(in.Storage.Folder)
	out.VCPUs = types.Int64Value(int64(in.CPU.VCPUs))
	out.MemorySizeMB = types.Int64Value(int64(in.Memory.SizeMB))
	out.IsTemplate = boolValue(in.IsTemplate)
	if out.StorageID.IsNull() || out.StorageID.IsUnknown() || out.StorageID.ValueString() == "" {
		out.StorageID = stringValue(in.Storage.ID)
	}
	return out
}
