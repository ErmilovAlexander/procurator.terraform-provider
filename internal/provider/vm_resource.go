package provider

import (
	"context"
	"fmt"
	"time"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &VMResource{}
var _ resource.ResourceWithConfigure = &VMResource{}
var _ resource.ResourceWithImportState = &VMResource{}

type VMResource struct {
	client client.Client
}

type vmDiskModel struct {
	Size          types.Int64  `tfsdk:"size"`
	Source        types.String `tfsdk:"source"`
	StorageID     types.String `tfsdk:"storage_id"`
	DeviceType    types.String `tfsdk:"device_type"`
	Bus           types.String `tfsdk:"bus"`
	Target        types.String `tfsdk:"target"`
	BootOrder     types.Int64  `tfsdk:"boot_order"`
	ProvisionType types.String `tfsdk:"provision_type"`
	DiskMode      types.String `tfsdk:"disk_mode"`
	Create        types.Bool   `tfsdk:"create"`
	Attach        types.Bool   `tfsdk:"attach"`
	Detach        types.Bool   `tfsdk:"detach"`
	Remove        types.Bool   `tfsdk:"remove"`
	Resize        types.Bool   `tfsdk:"resize"`
	ReadOnly      types.Bool   `tfsdk:"read_only"`
}

type vmNICModel struct {
	Network   types.String `tfsdk:"network"`
	NetBridge types.String `tfsdk:"net_bridge"`
	MAC       types.String `tfsdk:"mac"`
	Target    types.String `tfsdk:"target"`
	Model     types.String `tfsdk:"model"`
	BootOrder types.Int64  `tfsdk:"boot_order"`
	VLAN      types.Int64  `tfsdk:"vlan"`
}

type vmVideoModel struct {
	Adapter  types.String `tfsdk:"adapter"`
	Displays types.Int64  `tfsdk:"displays"`
	MemoryMB types.Int64  `tfsdk:"memory_mb"`
}

type vmUSBControllerModel struct {
	Type types.String `tfsdk:"type"`
}

type vmInputDeviceModel struct {
	Type types.String `tfsdk:"type"`
	Bus  types.String `tfsdk:"bus"`
}

type vmSpiceModel struct {
	ImgCompression      types.String `tfsdk:"img_compression"`
	JpegCompression     types.String `tfsdk:"jpeg_compression"`
	ZlibGlzCompression  types.String `tfsdk:"zlib_glz_compression"`
	StreamingMode       types.String `tfsdk:"streaming_mode"`
	PlaybackCompression types.Bool   `tfsdk:"playback_compression"`
	FileTransfer        types.Bool   `tfsdk:"file_transfer"`
	Clipboard           types.Bool   `tfsdk:"clipboard"`
}

type vmRemoteConsoleModel struct {
	Type          types.String   `tfsdk:"type"`
	Port          types.Int64    `tfsdk:"port"`
	Keymap        types.String   `tfsdk:"keymap"`
	Password      types.String   `tfsdk:"password"`
	GuestOSLock   types.Bool     `tfsdk:"guest_os_lock"`
	LimitSessions types.Int64    `tfsdk:"limit_sessions"`
	Spice         []vmSpiceModel `tfsdk:"spice"`
}

type vmGuestToolsModel struct {
	Enabled          types.Bool `tfsdk:"enabled"`
	SynchronizedTime types.Bool `tfsdk:"synchronized_time"`
	Balloon          types.Bool `tfsdk:"balloon"`
}

type vmBootOptionsModel struct {
	Firmware    types.String `tfsdk:"firmware"`
	BootDelayMS types.Int64  `tfsdk:"boot_delay_ms"`
	BootMenu    types.Bool   `tfsdk:"boot_menu"`
}

type vmResourceModel struct {
	ID             types.String           `tfsdk:"id"`
	Start          types.Bool             `tfsdk:"start"`
	PowerState     types.String           `tfsdk:"power_state"`
	PowerForce     types.Bool             `tfsdk:"power_force"`
	TemplateID     types.String           `tfsdk:"template_id"`
	Name           types.String           `tfsdk:"name"`
	UUID           types.String           `tfsdk:"uuid"`
	Compatibility  types.String           `tfsdk:"compatibility"`
	GuestOSFamily  types.String           `tfsdk:"guest_os_family"`
	GuestOSVersion types.String           `tfsdk:"guest_os_version"`
	MachineType    types.String           `tfsdk:"machine_type"`
	StorageID      types.String           `tfsdk:"storage_id"`
	StorageFolder  types.String           `tfsdk:"storage_folder"`
	VCPUs          types.Int64            `tfsdk:"vcpus"`
	MaxVCPUs       types.Int64            `tfsdk:"max_vcpus"`
	CorePerSocket  types.Int64            `tfsdk:"core_per_socket"`
	CPUModel       types.String           `tfsdk:"cpu_model"`
	CPUHotplug     types.Bool             `tfsdk:"cpu_hotplug"`
	MemorySizeMB   types.Int64            `tfsdk:"memory_size_mb"`
	MemoryHotplug  types.Bool             `tfsdk:"memory_hotplug"`
	DiskDevices    []vmDiskModel          `tfsdk:"disk_devices"`
	NetworkDevices []vmNICModel           `tfsdk:"network_devices"`
	VideoCard      []vmVideoModel         `tfsdk:"video_card"`
	USBControllers []vmUSBControllerModel `tfsdk:"usb_controllers"`
	InputDevices   []vmInputDeviceModel   `tfsdk:"input_devices"`
	RemoteConsole  []vmRemoteConsoleModel `tfsdk:"remote_console"`
	GuestTools     []vmGuestToolsModel    `tfsdk:"guest_tools"`
	BootOptions    []vmBootOptionsModel   `tfsdk:"boot_options"`
	IsTemplate     types.Bool             `tfsdk:"is_template"`
}

func NewVMResource() resource.Resource { return &VMResource{} }

func (r *VMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *VMResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerData).client
}

func (r *VMResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VM through procurator.core Vms service.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"start":            schema.BoolAttribute{Optional: true},
			"power_state":      schema.StringAttribute{Optional: true, Computed: true},
			"power_force":      schema.BoolAttribute{Optional: true},
			"template_id":      schema.StringAttribute{Optional: true},
			"name":             schema.StringAttribute{Required: true},
			"uuid":             schema.StringAttribute{Computed: true},
			"compatibility":    schema.StringAttribute{Optional: true, Computed: true},
			"guest_os_family":  schema.StringAttribute{Optional: true, Computed: true},
			"guest_os_version": schema.StringAttribute{Optional: true, Computed: true},
			"machine_type":     schema.StringAttribute{Optional: true, Computed: true},
			"storage_id":       schema.StringAttribute{Required: true},
			"storage_folder":   schema.StringAttribute{Optional: true, Computed: true},
			"vcpus":            schema.Int64Attribute{Optional: true, Computed: true},
			"max_vcpus":        schema.Int64Attribute{Optional: true, Computed: true},
			"core_per_socket":  schema.Int64Attribute{Optional: true, Computed: true},
			"cpu_model":        schema.StringAttribute{Optional: true, Computed: true},
			"cpu_hotplug":      schema.BoolAttribute{Optional: true, Computed: true},
			"memory_size_mb":   schema.Int64Attribute{Optional: true, Computed: true},
			"memory_hotplug":   schema.BoolAttribute{Optional: true, Computed: true},
			"is_template":      schema.BoolAttribute{Optional: true, Computed: true},
		},
		Blocks: map[string]schema.Block{
			"disk_devices": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"size":           schema.Int64Attribute{Required: true},
				"source":         schema.StringAttribute{Optional: true},
				"storage_id":     schema.StringAttribute{Required: true},
				"device_type":    schema.StringAttribute{Required: true},
				"bus":            schema.StringAttribute{Optional: true},
				"target":         schema.StringAttribute{Optional: true},
				"boot_order":     schema.Int64Attribute{Optional: true},
				"provision_type": schema.StringAttribute{Optional: true},
				"disk_mode":      schema.StringAttribute{Optional: true},
				"create":         schema.BoolAttribute{Optional: true},
				"attach":         schema.BoolAttribute{Optional: true},
				"detach":         schema.BoolAttribute{Optional: true},
				"remove":         schema.BoolAttribute{Optional: true},
				"resize":         schema.BoolAttribute{Optional: true},
				"read_only":      schema.BoolAttribute{Optional: true},
			}}},
			"network_devices": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"network":    schema.StringAttribute{Optional: true},
				"net_bridge": schema.StringAttribute{Optional: true},
				"mac":        schema.StringAttribute{Optional: true},
				"target":     schema.StringAttribute{Optional: true},
				"model":      schema.StringAttribute{Optional: true},
				"boot_order": schema.Int64Attribute{Optional: true},
				"vlan":       schema.Int64Attribute{Optional: true},
			}}},
			"video_card": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"adapter":   schema.StringAttribute{Optional: true},
				"displays":  schema.Int64Attribute{Optional: true},
				"memory_mb": schema.Int64Attribute{Optional: true},
			}}},
			"usb_controllers": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{Optional: true},
			}}},
			"input_devices": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{Optional: true},
				"bus":  schema.StringAttribute{Optional: true},
			}}},
			"remote_console": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"type":           schema.StringAttribute{Optional: true},
					"port":           schema.Int64Attribute{Optional: true},
					"keymap":         schema.StringAttribute{Optional: true},
					"password":       schema.StringAttribute{Optional: true, Sensitive: true},
					"guest_os_lock":  schema.BoolAttribute{Optional: true},
					"limit_sessions": schema.Int64Attribute{Optional: true},
				},
				Blocks: map[string]schema.Block{
					"spice": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
						"img_compression":      schema.StringAttribute{Optional: true},
						"jpeg_compression":     schema.StringAttribute{Optional: true},
						"zlib_glz_compression": schema.StringAttribute{Optional: true},
						"streaming_mode":       schema.StringAttribute{Optional: true},
						"playback_compression": schema.BoolAttribute{Optional: true},
						"file_transfer":        schema.BoolAttribute{Optional: true},
						"clipboard":            schema.BoolAttribute{Optional: true},
					}}},
				},
			}},
			"guest_tools": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"enabled":           schema.BoolAttribute{Optional: true},
				"synchronized_time": schema.BoolAttribute{Optional: true},
				"balloon":           schema.BoolAttribute{Optional: true},
			}}},
			"boot_options": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"firmware":      schema.StringAttribute{Optional: true},
				"boot_delay_ms": schema.Int64Attribute{Optional: true},
				"boot_menu":     schema.BoolAttribute{Optional: true},
			}}},
		},
	}
}

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm := expandVM(plan)
	var taskID string
	var err error
	var task *client.Task
	var validatedVM *client.VM

	if !plan.TemplateID.IsNull() && !plan.TemplateID.IsUnknown() && plan.TemplateID.ValueString() != "" {
		tmpl, err := r.client.GetTemplate(ctx, plan.TemplateID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read template", err.Error())
			return
		}
		deployVM := mergeVMForUpdate(tmpl, vm)
		deployVM.IsTemplate = true
		deployVM.Name = plan.Name.ValueString()
		deployVM.DeploymentName = plan.Name.ValueString()
		taskID, err = r.client.DeployTemplate(ctx, &client.DeployTemplateRequest{
			TemplateID: plan.TemplateID.ValueString(),
			Name:       plan.Name.ValueString(),
			StorageID:  plan.StorageID.ValueString(),
			Start:      boolFrom(plan.Start),
			PVM:        deployVM,
		})
		if err != nil {
			resp.Diagnostics.AddError("Template deploy failed", err.Error())
			return
		}
	} else {
		if plan.VCPUs.IsNull() || plan.VCPUs.IsUnknown() {
			resp.Diagnostics.AddError("Missing required argument", "The argument \"vcpus\" is required when template_id is not set.")
			return
		}
		if plan.MemorySizeMB.IsNull() || plan.MemorySizeMB.IsUnknown() {
			resp.Diagnostics.AddError("Missing required argument", "The argument \"memory_size_mb\" is required when template_id is not set.")
			return
		}
		validation, err := r.client.ValidateVM(ctx, vm)
		if err != nil {
			resp.Diagnostics.AddError("VM validation failed", err.Error())
			return
		}
		if validation != nil && len(validation.Errors) > 0 {
			for _, e := range validation.Errors {
				resp.Diagnostics.AddError(fmt.Sprintf("Validation failed: %s", e.Field), e.Message)
			}
			return
		}
		validatedVM = vm
		if validation != nil && validation.VM != nil {
			validatedVM = validation.VM
		}

		taskID, err = r.client.CreateVM(ctx, &client.CreateVMRequest{Start: boolFrom(plan.Start), VM: validatedVM})
		if err != nil {
			resp.Diagnostics.AddError("VM create failed", err.Error())
			return
		}
	}
	task, err = r.client.WaitTask(ctx, taskID)
	if err != nil {
		tflog.Warn(ctx, "wait task failed after create, trying reconciliation", map[string]any{
			"task_id": taskID,
			"error":   err.Error(),
			"name":    plan.Name.ValueString(),
		})

		reconciled, recErr := r.findVMByName(ctx, plan.Name.ValueString())
		if recErr == nil && reconciled != nil {
			tflog.Info(ctx, "recovered VM after failed wait", map[string]any{
				"task_id": taskID,
				"vm_id":   reconciled.DeploymentName,
				"name":    reconciled.Name,
			})
			task = &client.Task{ID: taskID, Status: 2, CreatedID: reconciled.DeploymentName}
		} else {
			resp.Diagnostics.AddError("Task wait failed after create", err.Error())
			return
		}
	}

	//lookupID := firstNonEmpty(task.CreatedID, plan.Name.ValueString(), vm.DeploymentName, vm.Name)
	//created, err := r.client.GetVM(ctx, lookupID)
	created, err := r.readCreatedVM(ctx, task, plan, vm)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created VM", err.Error())
		return
	}
	desiredPower := desiredPowerState(plan)
	if desiredPower == "" && !plan.Start.IsNull() && !plan.Start.IsUnknown() && plan.Start.ValueBool() {
		desiredPower = "running"
	}
	if desiredPower != "" && desiredPower != created.PowerState {
		ptask, err := r.client.SetVMPowerState(ctx, created.DeploymentName, desiredPower, plan.PowerForce.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("VM power state change failed after create", err.Error())
			return
		}
		if ptask != "" {
			if _, err = r.client.WaitTask(ctx, ptask); err != nil {
				resp.Diagnostics.AddError("Task wait failed after power state change", err.Error())
				return
			}
		}
		//created, err = r.client.GetVM(ctx, lookupID)
		created, err = r.readCreatedVM(ctx, task, plan, vm)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read created VM after power change", err.Error())
			return
		}
	}
	baseState := plan
	state := mergeVMResourceState(baseState, *created)
	if !plan.TemplateID.IsNull() && !plan.TemplateID.IsUnknown() && plan.TemplateID.ValueString() != "" && len(plan.DiskDevices) == 0 {
		state.DiskDevices = plan.DiskDevices
	}
	state.Start = plan.Start
	state.PowerState = stringValue(firstNonEmpty(desiredPower, created.PowerState))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.GetVM(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read VM", err.Error())
		return
	}
	newState := mergeVMResourceState(state, *vm)
	newState.Start = state.Start
	if state.PowerState.IsNull() || state.PowerState.IsUnknown() || state.PowerState.ValueString() == "" {
		newState.PowerState = stringValue(vm.PowerState)
	} else {
		newState.PowerState = state.PowerState
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmResourceModel
	var prior vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desiredPower := desiredPowerState(plan)
	priorPower := prior.PowerState.ValueString()
	powerChanged := desiredPower != "" && desiredPower != priorPower
	configChanged := vmConfigChanged(plan, prior)

	if !configChanged && powerChanged {
		ptask, err := r.client.SetVMPowerState(ctx, prior.ID.ValueString(), desiredPower, plan.PowerForce.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("VM power state change failed", err.Error())
			return
		}
		if ptask != "" {
			if _, err = r.client.WaitTask(ctx, ptask); err != nil {
				resp.Diagnostics.AddError("Task wait failed after power state change", err.Error())
				return
			}
		}
		updated, err := r.client.GetVM(ctx, prior.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read VM after power state change", err.Error())
			return
		}
		newState := mergeVMResourceState(plan, *updated)
		newState.Start = plan.Start
		newState.PowerState = stringValue(firstNonEmpty(desiredPower, updated.PowerState))
		newState.PowerForce = plan.PowerForce
		resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
		return
	}

	actual, err := r.client.GetVM(ctx, prior.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read current VM before update", err.Error())
		return
	}

	desired := expandVM(plan)
	desired.DeploymentName = firstNonEmpty(prior.ID.ValueString(), desired.DeploymentName, desired.Name)
	candidate := mergeVMForUpdate(actual, desired)

	validation, err := r.client.ValidateVM(ctx, candidate)
	if err != nil {
		resp.Diagnostics.AddError("VM validation failed", err.Error())
		return
	}
	if validation != nil && len(validation.Errors) > 0 {
		for _, e := range validation.Errors {
			resp.Diagnostics.AddError(fmt.Sprintf("Validation failed: %s", e.Field), e.Message)
		}
		return
	}
	validatedVM := candidate
	if validation != nil && validation.VM != nil {
		validatedVM = validation.VM
	}
	validatedVM.DeploymentName = firstNonEmpty(validatedVM.DeploymentName, prior.ID.ValueString(), candidate.DeploymentName, desired.Name)

	taskID, err := r.client.UpdateVM(ctx, validatedVM)
	if err != nil {
		resp.Diagnostics.AddError("VM update failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after update", err.Error())
		return
	}
	lookup := firstNonEmpty(validatedVM.DeploymentName, prior.ID.ValueString(), desired.Name)
	updated, err := r.client.GetVM(ctx, lookup)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated VM", err.Error())
		return
	}
	if powerChanged && desiredPower != updated.PowerState {
		ptask, err := r.client.SetVMPowerState(ctx, lookup, desiredPower, plan.PowerForce.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("VM power state change failed after update", err.Error())
			return
		}
		if ptask != "" {
			if _, err = r.client.WaitTask(ctx, ptask); err != nil {
				resp.Diagnostics.AddError("Task wait failed after power state change", err.Error())
				return
			}
		}
		updated, err = r.client.GetVM(ctx, lookup)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read updated VM after power change", err.Error())
			return
		}
	}
	baseState := plan
	newState := mergeVMResourceState(baseState, *updated)
	newState.Start = plan.Start
	newState.PowerState = stringValue(firstNonEmpty(desiredPower, updated.PowerState))
	newState.PowerForce = plan.PowerForce
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	taskID, err := r.client.DeleteVM(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("VM delete failed", err.Error())
		return
	}
	if _, err = r.client.WaitTask(ctx, taskID); err != nil {
		resp.Diagnostics.AddError("Task wait failed after delete", err.Error())
		return
	}
}

func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mergeVMForUpdate(actual, desired *client.VM) *client.VM {
	if actual == nil {
		return desired
	}
	if desired == nil {
		vm := *actual
		return &vm
	}
	vm := *actual
	vm.DeploymentName = firstNonEmpty(actual.DeploymentName, desired.DeploymentName)
	vm.Name = firstNonEmpty(desired.Name, actual.Name)
	vm.UUID = firstNonEmpty(actual.UUID, desired.UUID)
	vm.Compatibility = firstNonEmpty(desired.Compatibility, actual.Compatibility)
	vm.GuestOSFamily = firstNonEmpty(desired.GuestOSFamily, actual.GuestOSFamily)
	vm.GuestOSVersion = firstNonEmpty(desired.GuestOSVersion, actual.GuestOSVersion)
	vm.MachineType = firstNonEmpty(desired.MachineType, actual.MachineType)
	vm.IsTemplate = desired.IsTemplate

	if desired.Storage.ID != "" {
		vm.Storage.ID = desired.Storage.ID
	}
	// preserve folder unless user explicitly set non-empty folder
	if desired.Storage.Folder != "" {
		vm.Storage.Folder = desired.Storage.Folder
	}

	if desired.CPU.VCPUs != 0 {
		vm.CPU.VCPUs = desired.CPU.VCPUs
	}
	if desired.CPU.MaxVCPUs != 0 {
		vm.CPU.MaxVCPUs = desired.CPU.MaxVCPUs
	}
	if desired.CPU.CorePerSocket != 0 {
		vm.CPU.CorePerSocket = desired.CPU.CorePerSocket
	}
	if desired.CPU.Model != "" {
		vm.CPU.Model = desired.CPU.Model
	}
	vm.CPU.Hotplug = desired.CPU.Hotplug
	if desired.CPU.Shares != 0 {
		vm.CPU.Shares = desired.CPU.Shares
	}
	if desired.Memory.SizeMB != 0 {
		vm.Memory.SizeMB = desired.Memory.SizeMB
	}
	vm.Memory.Hotplug = desired.Memory.Hotplug
	if desired.Memory.ReservationMB != 0 {
		vm.Memory.ReservationMB = desired.Memory.ReservationMB
	}

	// Only override optional blocks if explicitly configured in Terraform.
	if desired.VideoCard.Adapter != "" || desired.VideoCard.Displays != 0 || desired.VideoCard.MemoryMB != 0 {
		vm.VideoCard = desired.VideoCard
	}
	if len(desired.USBControllers) > 0 {
		vm.USBControllers = desired.USBControllers
	}
	if len(desired.InputDevices) > 0 {
		vm.InputDevices = desired.InputDevices
	}
	if desired.Options.RemoteConsole.Type != "" || desired.Options.RemoteConsole.Port != 0 || desired.Options.RemoteConsole.Keymap != "" || desired.Options.RemoteConsole.Password != "" || desired.Options.RemoteConsole.GuestOSLock || desired.Options.RemoteConsole.LimitSessions != 0 {
		vm.Options.RemoteConsole = desired.Options.RemoteConsole
	}
	if desired.Options.GuestTools.Enabled || desired.Options.GuestTools.SynchronizedTime || desired.Options.GuestTools.Balloon {
		vm.Options.GuestTools = desired.Options.GuestTools
	}
	if desired.Options.BootOptions.Firmware != "" || desired.Options.BootOptions.BootDelayMS != 0 || desired.Options.BootOptions.BootMenu {
		vm.Options.BootOptions = desired.Options.BootOptions
	}

	vm.DiskDevices = mergeUpdateDisks(actual.DiskDevices, desired.DiskDevices, desired.Storage.ID)
	vm.NetworkDevices = mergeUpdateNICs(actual.NetworkDevices, desired.NetworkDevices)
	applyVMDefaults(&vm)
	return &vm
}

func mergeUpdateDisks(actual, desired []client.DiskDevice, defaultStorageID string) []client.DiskDevice {
	result := make([]client.DiskDevice, 0, len(actual)+len(desired))
	used := map[int]bool{}
	findDesired := func(a client.DiskDevice) (int, *client.DiskDevice) {
		for i := range desired {
			if used[i] {
				continue
			}
			d := desired[i]
			if d.Target != "" && a.Target != "" && d.Target == a.Target {
				return i, &desired[i]
			}
		}
		for i := range desired {
			if used[i] {
				continue
			}
			d := desired[i]
			if d.DeviceType == a.DeviceType && d.Bus == a.Bus && d.BootOrder == a.BootOrder {
				return i, &desired[i]
			}
		}
		return -1, nil
	}
	for _, a := range actual {
		idx, d := findDesired(a)
		if d == nil {
			// preserve server-added devices such as cdrom unchanged
			result = append(result, a)
			continue
		}
		used[idx] = true
		m := a
		if d.Source != "" {
			m.Source = d.Source
		}
		if d.Size != 0 && d.Size != a.Size {
			m.Size = d.Size
			m.Resize = true
		}
		if d.Resize {
			m.Resize = true
		}
		if d.StorageID != "" {
			m.StorageID = d.StorageID
		} else if m.StorageID == "" {
			m.StorageID = defaultStorageID
		}
		if d.DeviceType != "" {
			m.DeviceType = d.DeviceType
		}
		if d.Bus != "" {
			m.Bus = d.Bus
		}
		if d.Target != "" {
			m.Target = d.Target
		}
		if d.BootOrder != 0 {
			m.BootOrder = d.BootOrder
		}
		if d.ProvisionType != "" {
			m.ProvisionType = d.ProvisionType
		}
		if d.DiskMode != "" {
			m.DiskMode = d.DiskMode
		}
		m.ReadOnly = d.ReadOnly
		m.Create = false
		m.Remove = d.Remove
		m.Attach = d.Attach
		m.Detach = d.Detach
		result = append(result, m)
	}
	for i, d := range desired {
		if used[i] {
			continue
		}
		m := d
		if m.StorageID == "" {
			m.StorageID = defaultStorageID
		}
		if m.Attach {
			m.Create = false
		} else {
			m.Create = true
		}
		if !m.Attach {
			m.Attach = false
		}
		m.Detach = false
		m.Remove = false
		if !m.Resize {
			m.Resize = false
		}
		result = append(result, m)
	}
	return result
}

func mergeUpdateNICs(actual, desired []client.NetworkDevice) []client.NetworkDevice {
	if len(desired) == 0 {
		return actual
	}
	result := make([]client.NetworkDevice, 0, max(len(actual), len(desired)))
	used := map[int]bool{}
	findDesired := func(a client.NetworkDevice) (int, *client.NetworkDevice) {
		for i := range desired {
			if used[i] {
				continue
			}
			d := desired[i]
			if d.Target != "" && a.Target != "" && d.Target == a.Target {
				return i, &desired[i]
			}
		}
		for i := range desired {
			if used[i] {
				continue
			}
			d := desired[i]
			if d.Network == a.Network {
				return i, &desired[i]
			}
		}
		return -1, nil
	}
	for _, a := range actual {
		idx, d := findDesired(a)
		if d == nil {
			result = append(result, a)
			continue
		}
		used[idx] = true
		m := a
		if d.Network != "" {
			m.Network = d.Network
		}
		if d.NetBridge != "" {
			m.NetBridge = d.NetBridge
		}
		if d.Target != "" {
			m.Target = d.Target
		}
		if d.Model != "" {
			m.Model = d.Model
		}
		if d.BootOrder != 0 {
			m.BootOrder = d.BootOrder
		}
		if d.VLAN != 0 {
			m.VLAN = d.VLAN
		}
		if d.MAC != "" {
			m.MAC = d.MAC
		}
		result = append(result, m)
	}
	for i, d := range desired {
		if used[i] {
			continue
		}
		result = append(result, d)
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func expandVM(in vmResourceModel) *client.VM {
	vm := &client.VM{
		DeploymentName: in.ID.ValueString(),
		Name:           in.Name.ValueString(),
		UUID:           in.UUID.ValueString(),
		Compatibility:  in.Compatibility.ValueString(),
		GuestOSFamily:  in.GuestOSFamily.ValueString(),
		GuestOSVersion: in.GuestOSVersion.ValueString(),
		MachineType:    in.MachineType.ValueString(),
		IsTemplate:     boolFrom(in.IsTemplate),
		Storage:        client.VMStorage{ID: in.StorageID.ValueString(), Folder: in.StorageFolder.ValueString()},
		CPU: client.VMCPU{
			VCPUs:          int32(in.VCPUs.ValueInt64()),
			MaxVCPUs:       int32(in.MaxVCPUs.ValueInt64()),
			CorePerSocket:  int32(in.CorePerSocket.ValueInt64()),
			Model:          in.CPUModel.ValueString(),
			ReservationMHz: 0,
			LimitMHz:       0,
			Shares:         2000,
			Hotplug:        boolFrom(in.CPUHotplug),
		},
		Memory: client.VMMemory{
			SizeMB:        int32(in.MemorySizeMB.ValueInt64()),
			Hotplug:       boolFrom(in.MemoryHotplug),
			ReservationMB: int32(in.MemorySizeMB.ValueInt64()),
			LimitMB:       0,
		},
	}

	for _, v := range in.VideoCard {
		vm.VideoCard = client.VMVideo{Adapter: v.Adapter.ValueString(), Displays: int32(v.Displays.ValueInt64()), MemoryMB: int32(v.MemoryMB.ValueInt64())}
		break
	}
	for _, u := range in.USBControllers {
		vm.USBControllers = append(vm.USBControllers, client.USBController{Type: u.Type.ValueString()})
	}
	for _, d := range in.InputDevices {
		vm.InputDevices = append(vm.InputDevices, client.InputDevice{Type: d.Type.ValueString(), Bus: d.Bus.ValueString()})
	}
	for _, rc := range in.RemoteConsole {
		vm.Options.RemoteConsole = client.RemoteConsole{
			Type:          rc.Type.ValueString(),
			Port:          int32(rc.Port.ValueInt64()),
			Keymap:        rc.Keymap.ValueString(),
			Password:      rc.Password.ValueString(),
			GuestOSLock:   boolFrom(rc.GuestOSLock),
			LimitSessions: int32(rc.LimitSessions.ValueInt64()),
		}
		for _, sp := range rc.Spice {
			vm.Options.RemoteConsole.Spice = client.Spice{
				ImgCompression:      sp.ImgCompression.ValueString(),
				JpegCompression:     sp.JpegCompression.ValueString(),
				ZlibGlzCompression:  sp.ZlibGlzCompression.ValueString(),
				StreamingMode:       sp.StreamingMode.ValueString(),
				PlaybackCompression: boolFrom(sp.PlaybackCompression),
				FileTransfer:        boolFrom(sp.FileTransfer),
				Clipboard:           boolFrom(sp.Clipboard),
			}
			break
		}
		break
	}
	for _, gt := range in.GuestTools {
		vm.Options.GuestTools = client.GuestTools{Enabled: boolFrom(gt.Enabled), SynchronizedTime: boolFrom(gt.SynchronizedTime), Balloon: boolFrom(gt.Balloon)}
		break
	}
	for _, bo := range in.BootOptions {
		vm.Options.BootOptions = client.BootOptions{Firmware: bo.Firmware.ValueString(), BootDelayMS: int32(bo.BootDelayMS.ValueInt64()), BootMenu: boolFrom(bo.BootMenu)}
		break
	}

	applyVMDefaults(vm)
	for _, d := range in.DiskDevices {
		vm.DiskDevices = append(vm.DiskDevices, client.DiskDevice{
			Size:          uint64(d.Size.ValueInt64()) * 1024,
			Source:        d.Source.ValueString(),
			StorageID:     d.StorageID.ValueString(),
			DeviceType:    d.DeviceType.ValueString(),
			Bus:           d.Bus.ValueString(),
			Target:        d.Target.ValueString(),
			BootOrder:     int32(d.BootOrder.ValueInt64()),
			ProvisionType: d.ProvisionType.ValueString(),
			DiskMode:      d.DiskMode.ValueString(),
			Create:        boolFrom(d.Create),
			Attach:        boolFrom(d.Attach),
			Detach:        boolFrom(d.Detach),
			Remove:        boolFrom(d.Remove),
			Resize:        boolFrom(d.Resize),
			ReadOnly:      boolFrom(d.ReadOnly),
		})
	}
	for _, n := range in.NetworkDevices {
		vm.NetworkDevices = append(vm.NetworkDevices, client.NetworkDevice{
			Network:   n.Network.ValueString(),
			NetBridge: n.NetBridge.ValueString(),
			MAC:       n.MAC.ValueString(),
			Target:    n.Target.ValueString(),
			Model:     n.Model.ValueString(),
			BootOrder: int32(n.BootOrder.ValueInt64()),
			VLAN:      int32(n.VLAN.ValueInt64()),
		})
	}
	return vm
}

func desiredPowerState(in vmResourceModel) string {
	if !in.PowerState.IsNull() && !in.PowerState.IsUnknown() {
		return in.PowerState.ValueString()
	}
	return ""
}

func defaultFirmware(guestOSFamily, machineType string) string {
	_ = machineType
	if guestOSFamily == "windows" {
		return "efi"
	}
	return "bios"
}

func applyVMDefaults(vm *client.VM) {
	if vm == nil {
		return
	}
	if vm.MachineType == "" || vm.MachineType == "q35" {
		vm.MachineType = "pc-q35-6.2"
	}
	if vm.CPU.MaxVCPUs <= 0 {
		vm.CPU.MaxVCPUs = vm.CPU.VCPUs
	}
	if vm.CPU.CorePerSocket <= 0 {
		vm.CPU.CorePerSocket = vm.CPU.VCPUs
	}
	if vm.CPU.Model == "" {
		vm.CPU.Model = "host-model"
	}
	if vm.CPU.Shares <= 0 {
		vm.CPU.Shares = 1000
	}
	if vm.Memory.ReservationMB <= 0 {
		vm.Memory.ReservationMB = vm.Memory.SizeMB
	}
	if vm.VideoCard.Adapter == "" {
		vm.VideoCard.Adapter = "qxl"
	}
	if vm.VideoCard.Displays <= 0 {
		vm.VideoCard.Displays = 1
	}
	if vm.VideoCard.MemoryMB <= 0 {
		vm.VideoCard.MemoryMB = 16
	}
	if len(vm.USBControllers) == 0 {
		vm.USBControllers = []client.USBController{{Type: "usb2"}}
	}
	for i := range vm.USBControllers {
		if vm.USBControllers[i].Type == "" {
			vm.USBControllers[i].Type = "usb2"
		}
	}
	if len(vm.InputDevices) == 0 {
		vm.InputDevices = []client.InputDevice{{Type: "tablet", Bus: "usb"}}
	}
	for i := range vm.InputDevices {
		if vm.InputDevices[i].Type == "" {
			vm.InputDevices[i].Type = "tablet"
		}
		if vm.InputDevices[i].Bus == "" {
			vm.InputDevices[i].Bus = "usb"
		}
	}
	if vm.Options.RemoteConsole.Type == "" {
		vm.Options.RemoteConsole.Type = "spice"
	}
	if vm.Options.RemoteConsole.Port == 0 {
		vm.Options.RemoteConsole.Port = -1
	}
	if vm.Options.RemoteConsole.Keymap == "" {
		vm.Options.RemoteConsole.Keymap = "en_US"
	}
	if vm.Options.RemoteConsole.Spice.ImgCompression == "" {
		vm.Options.RemoteConsole.Spice.ImgCompression = "auto_glz"
	}
	if vm.Options.RemoteConsole.Spice.JpegCompression == "" {
		vm.Options.RemoteConsole.Spice.JpegCompression = "auto"
	}
	if vm.Options.RemoteConsole.Spice.ZlibGlzCompression == "" {
		vm.Options.RemoteConsole.Spice.ZlibGlzCompression = "auto"
	}
	if vm.Options.RemoteConsole.Spice.StreamingMode == "" {
		vm.Options.RemoteConsole.Spice.StreamingMode = "all"
	}
	if !vm.Options.RemoteConsole.Spice.PlaybackCompression {
		vm.Options.RemoteConsole.Spice.PlaybackCompression = true
	}
	if !vm.Options.RemoteConsole.Spice.FileTransfer {
		vm.Options.RemoteConsole.Spice.FileTransfer = true
	}
	if !vm.Options.RemoteConsole.Spice.Clipboard {
		vm.Options.RemoteConsole.Spice.Clipboard = true
	}
	if !vm.Options.GuestTools.Enabled {
		vm.Options.GuestTools.Enabled = true
	}
	if !vm.Options.GuestTools.SynchronizedTime {
		vm.Options.GuestTools.SynchronizedTime = true
	}
	if vm.Options.BootOptions.Firmware == "" {
		vm.Options.BootOptions.Firmware = defaultFirmware(vm.GuestOSFamily, vm.MachineType)
	}
	if vm.Options.BootOptions.BootDelayMS == 0 {
		vm.Options.BootOptions.BootDelayMS = 1000
	}
	if !vm.Options.BootOptions.BootMenu {
		vm.Options.BootOptions.BootMenu = true
	}
	for i := range vm.DiskDevices {
		if vm.DiskDevices[i].StorageID == "" {
			vm.DiskDevices[i].StorageID = vm.Storage.ID
		}
		if vm.DiskDevices[i].DeviceType == "" {
			vm.DiskDevices[i].DeviceType = "disk"
		}
		if vm.DiskDevices[i].ProvisionType == "" {
			vm.DiskDevices[i].ProvisionType = "thick"
		}
		if vm.DiskDevices[i].DiskMode == "" {
			vm.DiskDevices[i].DiskMode = "dependent"
		}
	}
	for i := range vm.NetworkDevices {
		if vm.NetworkDevices[i].Model == "" {
			vm.NetworkDevices[i].Model = "rtl8139"
		}
	}
}

func flattenVMResource(in client.VM) vmResourceModel {
	res := vmResourceModel{
		ID:             stringValue(firstNonEmpty(in.DeploymentName, in.Name)),
		Name:           stringValue(in.Name),
		PowerState:     stringValue(in.PowerState),
		UUID:           stringValue(in.UUID),
		Compatibility:  stringValue(in.Compatibility),
		GuestOSFamily:  stringValue(in.GuestOSFamily),
		GuestOSVersion: stringValue(in.GuestOSVersion),
		MachineType:    stringValue(in.MachineType),
		StorageID:      stringValue(in.Storage.ID),
		StorageFolder:  stringValue(in.Storage.Folder),
		VCPUs:          types.Int64Value(int64(in.CPU.VCPUs)),
		MaxVCPUs:       int64Value(int64(in.CPU.MaxVCPUs)),
		CorePerSocket:  int64Value(int64(in.CPU.CorePerSocket)),
		CPUModel:       stringValue(in.CPU.Model),
		CPUHotplug:     boolValue(in.CPU.Hotplug),
		MemorySizeMB:   types.Int64Value(int64(in.Memory.SizeMB)),
		MemoryHotplug:  boolValue(in.Memory.Hotplug),
		IsTemplate:     boolValue(in.IsTemplate),
	}
	res.VideoCard = []vmVideoModel{{Adapter: stringValue(in.VideoCard.Adapter), Displays: int64Value(int64(in.VideoCard.Displays)), MemoryMB: int64Value(int64(in.VideoCard.MemoryMB))}}
	for _, u := range in.USBControllers {
		res.USBControllers = append(res.USBControllers, vmUSBControllerModel{Type: stringValue(u.Type)})
	}
	for _, d := range in.InputDevices {
		res.InputDevices = append(res.InputDevices, vmInputDeviceModel{Type: stringValue(d.Type), Bus: stringValue(d.Bus)})
	}
	res.RemoteConsole = []vmRemoteConsoleModel{{
		Type:          stringValue(in.Options.RemoteConsole.Type),
		Port:          int64Value(int64(in.Options.RemoteConsole.Port)),
		Keymap:        stringValue(in.Options.RemoteConsole.Keymap),
		Password:      stringValue(in.Options.RemoteConsole.Password),
		GuestOSLock:   boolValue(in.Options.RemoteConsole.GuestOSLock),
		LimitSessions: int64Value(int64(in.Options.RemoteConsole.LimitSessions)),
		Spice: []vmSpiceModel{{
			ImgCompression:      stringValue(in.Options.RemoteConsole.Spice.ImgCompression),
			JpegCompression:     stringValue(in.Options.RemoteConsole.Spice.JpegCompression),
			ZlibGlzCompression:  stringValue(in.Options.RemoteConsole.Spice.ZlibGlzCompression),
			StreamingMode:       stringValue(in.Options.RemoteConsole.Spice.StreamingMode),
			PlaybackCompression: boolValue(in.Options.RemoteConsole.Spice.PlaybackCompression),
			FileTransfer:        boolValue(in.Options.RemoteConsole.Spice.FileTransfer),
			Clipboard:           boolValue(in.Options.RemoteConsole.Spice.Clipboard),
		}},
	}}
	res.GuestTools = []vmGuestToolsModel{{Enabled: boolValue(in.Options.GuestTools.Enabled), SynchronizedTime: boolValue(in.Options.GuestTools.SynchronizedTime), Balloon: boolValue(in.Options.GuestTools.Balloon)}}
	res.BootOptions = []vmBootOptionsModel{{Firmware: stringValue(in.Options.BootOptions.Firmware), BootDelayMS: int64Value(int64(in.Options.BootOptions.BootDelayMS)), BootMenu: boolValue(in.Options.BootOptions.BootMenu)}}
	for _, d := range in.DiskDevices {
		res.DiskDevices = append(res.DiskDevices, vmDiskModel{Size: types.Int64Value(int64(d.Size)), Source: stringValue(d.Source), StorageID: stringValue(d.StorageID), DeviceType: stringValue(d.DeviceType), Bus: stringValue(d.Bus), Target: stringValue(d.Target), BootOrder: int64Value(int64(d.BootOrder)), ProvisionType: stringValue(d.ProvisionType), DiskMode: stringValue(d.DiskMode), Create: boolValue(d.Create), Attach: boolValue(d.Attach), Detach: boolValue(d.Detach), Remove: boolValue(d.Remove), Resize: boolValue(d.Resize), ReadOnly: boolValue(d.ReadOnly)})
	}
	for _, n := range in.NetworkDevices {
		res.NetworkDevices = append(res.NetworkDevices, vmNICModel{Network: stringValue(n.Network), NetBridge: stringValue(n.NetBridge), MAC: stringValue(n.MAC), Target: stringValue(n.Target), Model: stringValue(n.Model), BootOrder: int64Value(int64(n.BootOrder)), VLAN: int64Value(int64(n.VLAN))})
	}
	return res
}

func normalizeManagedState(in vmResourceModel) vmResourceModel {
	in.VideoCard = normalizeVideoCardState(in.VideoCard)
	in.USBControllers = normalizeUSBControllersState(in.USBControllers)
	in.InputDevices = normalizeInputDevicesState(in.InputDevices)
	in.RemoteConsole = normalizeRemoteConsoleState(in.RemoteConsole)
	in.GuestTools = normalizeGuestToolsState(in.GuestTools)
	in.BootOptions = normalizeBootOptionsState(in.BootOptions)
	in.DiskDevices = normalizeDiskDevicesState(in.DiskDevices)
	return in
}

func normalizeVideoCardState(v []vmVideoModel) []vmVideoModel {
	if len(v) == 0 {
		return v
	}
	if len(v) == 1 {
		x := v[0]
		adapter := x.Adapter.ValueString()
		displays := x.Displays.ValueInt64()
		mem := x.MemoryMB.ValueInt64()
		if (adapter == "" || adapter == "qxl") && (displays == 0 || displays == 1) && (mem == 0 || mem == 16) {
			return nil
		}
	}
	return v
}

func normalizeUSBControllersState(v []vmUSBControllerModel) []vmUSBControllerModel {
	if len(v) == 0 {
		return v
	}
	allDefault := true
	for _, x := range v {
		if t := x.Type.ValueString(); t != "" && t != "usb2" {
			allDefault = false
			break
		}
	}
	if allDefault {
		return nil
	}
	return v
}

func normalizeInputDevicesState(v []vmInputDeviceModel) []vmInputDeviceModel {
	if len(v) == 0 {
		return v
	}
	allDefault := true
	for _, x := range v {
		t := x.Type.ValueString()
		b := x.Bus.ValueString()
		if !((t == "" || t == "tablet") && (b == "" || b == "usb")) {
			allDefault = false
			break
		}
	}
	if allDefault {
		return nil
	}
	return v
}

func normalizeRemoteConsoleState(v []vmRemoteConsoleModel) []vmRemoteConsoleModel {
	if len(v) == 0 {
		return v
	}
	if len(v) == 1 {
		r := v[0]
		if (r.Type.ValueString() == "" || r.Type.ValueString() == "spice") &&
			(r.Port.ValueInt64() == 0 || r.Port.ValueInt64() == -1) &&
			(r.Keymap.ValueString() == "" || r.Keymap.ValueString() == "en_US") {
			return nil
		}
	}
	return v
}

func normalizeGuestToolsState(v []vmGuestToolsModel) []vmGuestToolsModel {
	if len(v) == 0 {
		return v
	}
	if len(v) == 1 {
		g := v[0]
		if (!g.Enabled.IsNull() && g.Enabled.ValueBool()) && (!g.SynchronizedTime.IsNull() && g.SynchronizedTime.ValueBool()) &&
			(g.Balloon.IsNull() || !g.Balloon.ValueBool()) {
			return nil
		}
	}
	return v
}

func normalizeBootOptionsState(v []vmBootOptionsModel) []vmBootOptionsModel {
	if len(v) == 0 {
		return v
	}
	if len(v) == 1 {
		b := v[0]
		fw := b.Firmware.ValueString()
		if (fw == "" || fw == "bios" || fw == "efi") && (b.BootDelayMS.ValueInt64() == 0 || b.BootDelayMS.ValueInt64() == 1000) &&
			(b.BootMenu.IsNull() || b.BootMenu.ValueBool()) {
			return nil
		}
	}
	return v
}

func normalizeDiskDevicesState(v []vmDiskModel) []vmDiskModel {
	if len(v) == 0 {
		return v
	}
	out := make([]vmDiskModel, 0, len(v))
	for _, d := range v {
		t := d.DeviceType.ValueString()
		if t != "" && t != "disk" {
			continue
		}
		out = append(out, d)
	}
	return out
}

func mergeVMResourceState(base vmResourceModel, in client.VM) vmResourceModel {
	actual := normalizeManagedState(flattenVMResource(in))
	merged := base

	merged.ID = actual.ID
	merged.UUID = actual.UUID
	merged.IsTemplate = actual.IsTemplate
	if merged.PowerState.IsNull() || merged.PowerState.IsUnknown() || merged.PowerState.ValueString() == "" {
		merged.PowerState = actual.PowerState
	}

	if merged.Name.IsNull() || merged.Name.IsUnknown() || merged.Name.ValueString() == "" {
		merged.Name = actual.Name
	}
	if merged.Compatibility.IsNull() || merged.Compatibility.IsUnknown() || merged.Compatibility.ValueString() == "" {
		merged.Compatibility = actual.Compatibility
	}
	if merged.GuestOSFamily.IsNull() || merged.GuestOSFamily.IsUnknown() || merged.GuestOSFamily.ValueString() == "" {
		merged.GuestOSFamily = actual.GuestOSFamily
	}
	if merged.GuestOSVersion.IsNull() || merged.GuestOSVersion.IsUnknown() || merged.GuestOSVersion.ValueString() == "" {
		merged.GuestOSVersion = actual.GuestOSVersion
	}
	if merged.MachineType.IsNull() || merged.MachineType.IsUnknown() || merged.MachineType.ValueString() == "" {
		merged.MachineType = actual.MachineType
	}
	if merged.StorageID.IsNull() || merged.StorageID.IsUnknown() || merged.StorageID.ValueString() == "" {
		merged.StorageID = actual.StorageID
	}
	if merged.StorageFolder.IsNull() || merged.StorageFolder.IsUnknown() {
		merged.StorageFolder = actual.StorageFolder
	}
	if merged.VCPUs.IsNull() || merged.VCPUs.IsUnknown() || merged.VCPUs.ValueInt64() == 0 {
		merged.VCPUs = actual.VCPUs
	}
	if merged.MaxVCPUs.IsNull() || merged.MaxVCPUs.IsUnknown() || merged.MaxVCPUs.ValueInt64() == 0 {
		merged.MaxVCPUs = actual.MaxVCPUs
	}
	if merged.CorePerSocket.IsNull() || merged.CorePerSocket.IsUnknown() || merged.CorePerSocket.ValueInt64() == 0 {
		merged.CorePerSocket = actual.CorePerSocket
	}
	if merged.CPUModel.IsNull() || merged.CPUModel.IsUnknown() || merged.CPUModel.ValueString() == "" {
		merged.CPUModel = actual.CPUModel
	}
	if merged.CPUHotplug.IsNull() || merged.CPUHotplug.IsUnknown() {
		merged.CPUHotplug = actual.CPUHotplug
	}
	if merged.MemorySizeMB.IsNull() || merged.MemorySizeMB.IsUnknown() || merged.MemorySizeMB.ValueInt64() == 0 {
		merged.MemorySizeMB = actual.MemorySizeMB
	}
	if merged.MemoryHotplug.IsNull() || merged.MemoryHotplug.IsUnknown() {
		merged.MemoryHotplug = actual.MemoryHotplug
	}
	// Keep optional nested blocks exactly as configured. The provider may use
	// internal defaults when talking to Procurator, but must not invent blocks
	// in Terraform state if the user did not declare them in configuration.
	if len(base.DiskDevices) == 0 {
		merged.DiskDevices = nil
	} else if len(actual.DiskDevices) >= len(merged.DiskDevices) {
		for i := range merged.DiskDevices {
			if merged.DiskDevices[i].Target.IsNull() || merged.DiskDevices[i].Target.IsUnknown() || merged.DiskDevices[i].Target.ValueString() == "" {
				merged.DiskDevices[i].Target = actual.DiskDevices[i].Target
			}
		}
	}

	if len(base.NetworkDevices) == 0 {
		merged.NetworkDevices = nil
	} else {
		for i := range merged.NetworkDevices {
			if i < len(actual.NetworkDevices) {
				if merged.NetworkDevices[i].NetBridge.IsNull() || merged.NetworkDevices[i].NetBridge.IsUnknown() {
					merged.NetworkDevices[i].NetBridge = actual.NetworkDevices[i].NetBridge
				}
				// MAC is runtime/server-generated, do not persist it into managed config state.
				merged.NetworkDevices[i].MAC = types.StringNull()
			}
		}
	}
	if len(base.VideoCard) == 0 {
		merged.VideoCard = nil
	}
	if len(base.USBControllers) == 0 {
		merged.USBControllers = nil
	}
	if len(base.InputDevices) == 0 {
		merged.InputDevices = nil
	}
	if len(base.RemoteConsole) == 0 {
		merged.RemoteConsole = nil
	}
	if len(base.GuestTools) == 0 {
		merged.GuestTools = nil
	}
	if len(base.BootOptions) == 0 {
		merged.BootOptions = nil
	}

	return merged
}

func vmConfigChanged(plan vmResourceModel, prior vmResourceModel) bool {
	if !plan.TemplateID.Equal(prior.TemplateID) || !plan.Name.Equal(prior.Name) || !plan.StorageID.Equal(prior.StorageID) || !plan.StorageFolder.Equal(prior.StorageFolder) || !plan.MachineType.Equal(prior.MachineType) || !plan.VCPUs.Equal(prior.VCPUs) || !plan.MaxVCPUs.Equal(prior.MaxVCPUs) || !plan.CorePerSocket.Equal(prior.CorePerSocket) || !plan.CPUModel.Equal(prior.CPUModel) || !plan.CPUHotplug.Equal(prior.CPUHotplug) || !plan.MemorySizeMB.Equal(prior.MemorySizeMB) || !plan.MemoryHotplug.Equal(prior.MemoryHotplug) || !plan.Start.Equal(prior.Start) {
		return true
	}
	if len(plan.DiskDevices) != len(prior.DiskDevices) || len(plan.NetworkDevices) != len(prior.NetworkDevices) || len(plan.GuestTools) != len(prior.GuestTools) || len(plan.BootOptions) != len(prior.BootOptions) {
		return true
	}
	for i := range plan.DiskDevices {
		if plan.DiskDevices[i] != prior.DiskDevices[i] {
			return true
		}
	}
	for i := range plan.NetworkDevices {
		if plan.NetworkDevices[i] != prior.NetworkDevices[i] {
			return true
		}
	}
	for i := range plan.GuestTools {
		if plan.GuestTools[i] != prior.GuestTools[i] {
			return true
		}
	}
	for i := range plan.BootOptions {
		if plan.BootOptions[i] != prior.BootOptions[i] {
			return true
		}
	}
	return false
}

func (r *VMResource) findVMByName(ctx context.Context, name string) (*client.VM, error) {
	if name == "" {
		return nil, nil
	}
	vms, err := r.client.ListVMs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].Name == name || vms[i].DeploymentName == name {
			vm := vms[i]
			return &vm, nil
		}
	}
	return nil, nil
}

func (r *VMResource) readCreatedVM(ctx context.Context, task *client.Task, plan vmResourceModel, vm *client.VM) (*client.VM, error) {
	candidates := []string{
		firstNonEmpty(task.CreatedID),
		firstNonEmpty(plan.Name.ValueString()),
		firstNonEmpty(vm.DeploymentName),
		firstNonEmpty(vm.Name),
	}

	for i := 0; i < 10; i++ {
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			found, err := r.client.GetVM(ctx, candidate)
			if err == nil && found != nil {
				return found, nil
			}
		}

		reconciled, err := r.findVMByName(ctx, plan.Name.ValueString())
		if err == nil && reconciled != nil {
			return reconciled, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("not found")
}
