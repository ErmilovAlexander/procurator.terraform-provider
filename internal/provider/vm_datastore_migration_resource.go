package provider

import (
	"context"
	"fmt"
	"sort"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
)

var _ resource.Resource = &VMDatastoreMigrationResource{}
var _ resource.ResourceWithConfigure = &VMDatastoreMigrationResource{}

type VMDatastoreMigrationResource struct{ client client.Client }

type vmDatastoreMigrationResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	VMID              types.String   `tfsdk:"vm_id"`
	TargetDatastoreID types.String   `tfsdk:"target_datastore_id"`
	IncludeMeta       types.Bool     `tfsdk:"include_meta"`
	DiskSourcePaths   types.List     `tfsdk:"disk_source_paths"`
	TaskID            types.String   `tfsdk:"task_id"`
	SourceDatastoreID types.String   `tfsdk:"source_datastore_id"`
	FinalDatastoreID  types.String   `tfsdk:"final_datastore_id"`
}

func NewVMDatastoreMigrationResource() resource.Resource { return &VMDatastoreMigrationResource{} }

func (r *VMDatastoreMigrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_datastore_migration"
}

func (r *VMDatastoreMigrationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}

func (r *VMDatastoreMigrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Moves VM files between datastores through procurator.core task vm.migrate_datastore. By default migrates VM metadata and all attached disks to the target datastore.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"vm_id":               schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"target_datastore_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"include_meta":        schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"disk_source_paths": schema.ListAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				Description:   "Specific VM disk source paths to migrate. When omitted, provider migrates all VM disks from the current VM description.",
			},
			"task_id":             schema.StringAttribute{Computed: true},
			"source_datastore_id": schema.StringAttribute{Computed: true},
			"final_datastore_id":  schema.StringAttribute{Computed: true},
		},
	}
}

func (r *VMDatastoreMigrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmDatastoreMigrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.GetVM(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("VM datastore migration failed", fmt.Sprintf("failed to read VM %q before migration: %v", plan.VMID.ValueString(), err))
		return
	}

	includeMeta := true
	if !plan.IncludeMeta.IsNull() && !plan.IncludeMeta.IsUnknown() {
		includeMeta = plan.IncludeMeta.ValueBool()
	}
	diskPaths, err := selectedMigrationDiskPaths(ctx, plan.DiskSourcePaths, vm)
	if err != nil {
		resp.Diagnostics.AddError("VM datastore migration failed", err.Error())
		return
	}

	items := make(map[string]client.VMDatastoreMigrationItem, len(diskPaths)+1)
	if includeMeta {
		items["meta"] = client.VMDatastoreMigrationItem{ID: plan.TargetDatastoreID.ValueString()}
	}
	for _, src := range diskPaths {
		items[src] = client.VMDatastoreMigrationItem{ID: plan.TargetDatastoreID.ValueString()}
	}

	taskID, err := r.client.MigrateVMDatastore(ctx, plan.VMID.ValueString(), items)
	if err != nil {
		resp.Diagnostics.AddError("VM datastore migration failed", err.Error())
		return
	}
	if _, err := r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after VM datastore migration", err.Error())
		return
	}

	vmAfter, err := r.client.GetVM(ctx, plan.VMID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VM after datastore migration", err.Error())
		return
	}

	state := vmDatastoreMigrationResourceModel{
		ID:                stringValue(fmt.Sprintf("%s:%s", plan.VMID.ValueString(), plan.TargetDatastoreID.ValueString())),
		VMID:              plan.VMID,
		TargetDatastoreID: plan.TargetDatastoreID,
		IncludeMeta:       boolValue(includeMeta),
		DiskSourcePaths:   stringListValue(diskPaths),
		TaskID:            stringValue(taskID),
		SourceDatastoreID: stringValue(vm.Storage.ID),
		FinalDatastoreID:  stringValue(vmAfter.Storage.ID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMDatastoreMigrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmDatastoreMigrationResourceModel
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
		resp.Diagnostics.AddError("Failed to read VM after datastore migration", err.Error())
		return
	}
	state.FinalDatastoreID = stringValue(vm.Storage.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMDatastoreMigrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state vmDatastoreMigrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMDatastoreMigrationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No remote delete operation. Removing the resource only forgets the completed migration action from Terraform state.
}

func selectedMigrationDiskPaths(ctx context.Context, plan types.List, vm *client.VM) ([]string, error) {
	if vm == nil {
		return nil, fmt.Errorf("vm is nil")
	}
	if !plan.IsNull() && !plan.IsUnknown() {
		var vals []string
		if diags := plan.ElementsAs(ctx, &vals, false); diags.HasError() {
			return nil, fmt.Errorf("failed to decode disk_source_paths")
		}
		out := make([]string, 0, len(vals))
		seen := map[string]struct{}{}
		for _, s := range vals {
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("disk_source_paths resolved to an empty list")
		}
		sort.Strings(out)
		return out, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(vm.DiskDevices))
	for _, disk := range vm.DiskDevices {
		if disk.Source == "" {
			continue
		}
		if _, ok := seen[disk.Source]; ok {
			continue
		}
		seen[disk.Source] = struct{}{}
		out = append(out, disk.Source)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("vm %q has no disk source paths to migrate", firstNonEmpty(vm.DeploymentName, vm.Name, vm.UUID))
	}
	sort.Strings(out)
	return out, nil
}

func stringListValue(in []string) types.List {
	if len(in) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	vals := make([]attr.Value, 0, len(in))
	for _, v := range in {
		vals = append(vals, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, vals)
}
