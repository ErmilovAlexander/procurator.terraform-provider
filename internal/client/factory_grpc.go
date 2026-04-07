package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	grpc_reflection_v1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	corePackage = "procurator.core.api"
)

type grpcClient struct {
	cfg      Config
	conn     *grpc.ClientConn
	filesMu  sync.Mutex
	files    *protoregistry.Files
	messages map[string]protoreflect.MessageDescriptor
}

func New(cfg Config) (Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	transportCreds, err := grpcTransportCredentials(cfg)
	if err != nil {
		return nil, err
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithUnaryInterceptor(authUnaryInterceptor(cfg.Token)),
		grpc.WithStreamInterceptor(authStreamInterceptor(cfg.Token)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	if cfg.Authority != "" {
		dialOpts = append(dialOpts, grpc.WithAuthority(cfg.Authority))
	}

	conn, err := grpc.Dial(cfg.Endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	c := &grpcClient{
		cfg:      cfg,
		conn:     conn,
		files:    new(protoregistry.Files),
		messages: map[string]protoreflect.MessageDescriptor{},
	}

	if cfg.Token == "" && cfg.Username != "" {
		token, err := c.login(context.Background(), cfg.Username, cfg.Password)
		if err != nil {
			return nil, err
		}
		c.cfg.Token = token
	}

	return c, nil
}

func grpcTransportCredentials(cfg Config) (credentials.TransportCredentials, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.Authority != "" {
		tlsCfg.ServerName = cfg.Authority
	}

	switch {
	case cfg.CAFile != "":
		pemData, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca_file %q: %w", cfg.CAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("parse ca_file %q: no certificates found", cfg.CAFile)
		}
		tlsCfg.RootCAs = pool

	case len(embeddedRootCA) > 0:
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(embeddedRootCA) {
			return nil, fmt.Errorf("parse embedded CA: no certificates found")
		}
		tlsCfg.RootCAs = pool

	case cfg.Insecure:
		// Разрешаем сборку/запуск без CA только в insecure режиме.
		// RootCAs не задаём, ниже будет InsecureSkipVerify.
	default:
		return nil, fmt.Errorf("ca_file is required when provider binary is built without embedded CA")
	}

	if cfg.Insecure {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	}

	return credentials.NewTLS(tlsCfg), nil
}

func authUnaryInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", bearer(token))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func authStreamInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", bearer(token))
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func bearer(token string) string {
	if token == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}

func (c *grpcClient) login(ctx context.Context, username, password string) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Auth", "Login"), map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	token := getString(resp, "access_token")
	if token == "" {
		return "", fmt.Errorf("empty access_token returned by Auth.Login")
	}
	return token, nil
}

func (c *grpcClient) ListTemplates(ctx context.Context) ([]VM, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "ListTemplates2"), map[string]any{"page": 1, "page_size": 1000})
	if err != nil {
		return nil, err
	}
	items := getList(resp, "items")
	out := make([]VM, 0, len(items))
	for _, item := range items {
		out = append(out, flattenVM(item))
	}
	return out, nil
}

func (c *grpcClient) GetTemplate(ctx context.Context, target string) (*VM, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "GetTemplate2"), map[string]any{"target_id": target})
	if err != nil {
		return nil, err
	}
	vm := flattenVM(resp)
	return &vm, nil
}

func (c *grpcClient) ConvertVMToTemplate(ctx context.Context, vmID string) (string, error) {
	if vmID == "" {
		return "", fmt.Errorf("vm id is required")
	}
	resp, err := c.unary(ctx, fullMethod("Vms", "ConvertVmToTemplate2"), map[string]any{
		"target_id": vmID,
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) ConvertTemplateToVM(ctx context.Context, templateID string) (string, error) {
	if templateID == "" {
		return "", fmt.Errorf("template id is required")
	}
	resp, err := c.unary(ctx, fullMethod("Vms", "ConvertTemplateToVm2"), map[string]any{
		"target_id": templateID,
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) CreateTemplateFromVM(ctx context.Context, vmID, storageID, name string) (string, error) {
	// Use the non-deprecated v2 flow. In procurator.core v1.11.4
	// ConvertVmToTemplate2 converts an existing powered-off VM to a template in-place
	// and returns an ActionResponse with task_id.
	// The resulting template keeps the same deployment identity, so we return vmID
	// after waiting for the task.
	resp, err := c.unary(ctx, fullMethod("Vms", "ConvertVmToTemplate2"), map[string]any{
		"target_id": vmID,
	})
	if err != nil {
		return "", err
	}
	taskID := getString(resp, "task_id")
	if taskID != "" {
		if _, err := c.WaitTask(ctx, taskID); err != nil {
			return "", err
		}
	}
	return vmID, nil
}

func (c *grpcClient) DeleteTemplate(ctx context.Context, id string) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "RemoveTemplate2"), map[string]any{"target_id": id})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) DeployTemplate(ctx context.Context, req *DeployTemplateRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("deploy template request is nil")
	}
	if req.TemplateID == "" {
		return "", fmt.Errorf("template_id is required")
	}
	tmpl, err := c.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"req": map[string]any{
			"id":          req.TemplateID,
			"storage_dst": req.StorageID,
			"vm_name":     req.Name,
			"start":       req.Start,
			"clone_count": req.CloneCount,
			"linked":      req.Linked,
		},
		"pvm": encodeVM(tmpl),
	}
	resp, err := c.unary(ctx, fullMethod("Vms", "DeployTemplateDirect"), payload)
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) GetHost(ctx context.Context) (*Host, error) {
	resp, err := c.unary(ctx, fullMethod("Host", "Get"), nil)
	if err != nil {
		return nil, err
	}
	return &Host{
		ID:       firstNonEmpty(getString(resp, "host_uuid"), getString(resp, "hostname")),
		Name:     getString(resp, "hostname"),
		Hostname: getString(resp, "hostname"),
		UUID:     getString(resp, "host_uuid"),
		Vendor:   getString(resp, "processor_type"),
		Model:    getString(resp, "model"),
		Version:  nestedString(resp, "system_information", "bios_version"),
	}, nil
}

func (c *grpcClient) ListDatastores(ctx context.Context) ([]Datastore, error) {
	resp, err := c.unary(ctx, fullMethod("Datastores", "List"), nil)
	if err != nil {
		return nil, err
	}
	list := getList(resp, "datastores")
	out := make([]Datastore, 0, len(list))
	for _, item := range list {
		out = append(out, flattenDatastore(item))
	}
	return out, nil
}

func (c *grpcClient) GetDatastore(ctx context.Context, id string) (*Datastore, error) {
	resp, err := c.unary(ctx, fullMethod("Datastores", "Get"), map[string]any{"target_id": id})
	if err != nil {
		return nil, err
	}
	ds := flattenDatastore(resp)
	return &ds, nil
}

func (c *grpcClient) CreateDatastore(ctx context.Context, ds *Datastore) (string, error) {
	if ds == nil {
		return "", fmt.Errorf("datastore is nil")
	}
	req := map[string]any{
		"name":      ds.Name,
		"type_code": ds.TypeCode,
	}
	if ds.Server != "" {
		req["server"] = ds.Server
	}
	if ds.Folder != "" {
		req["folder"] = ds.Folder
	}
	if ds.Readonly {
		req["readonly"] = ds.Readonly
	}
	if len(ds.Devices) > 0 {
		arr := make([]any, 0, len(ds.Devices))
		for _, d := range ds.Devices {
			arr = append(arr, d)
		}
		req["devices"] = arr
	}
	if ds.Reinit != nil {
		req["reinit"] = *ds.Reinit
	}
	if ds.NConnect != nil {
		req["nconnect"] = *ds.NConnect
	}
	resp, err := c.unary(ctx, fullMethod("Datastores", "Create"), req)
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) DeleteDatastore(ctx context.Context, id string) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Datastores", "Delete"), map[string]any{"target_id": id})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) BrowseDatastoreItem(ctx context.Context, path string) (*DatastoreItem, error) {
	resp, err := c.unary(ctx, fullMethod("Datastores", "BrowseFolder"), map[string]any{"path": path})
	if err != nil {
		return nil, err
	}
	item := flattenDatastoreItem(resp)
	return &item, nil
}

func (c *grpcClient) CreateDatastoreFolder(ctx context.Context, path string) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Datastores", "NewFolder"), map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) DeleteDatastoreItem(ctx context.Context, paths []string) (string, error) {
	arr := make([]any, 0, len(paths))
	for _, p := range paths {
		arr = append(arr, p)
	}
	resp, err := c.unary(ctx, fullMethod("Datastores", "DeleteItem"), map[string]any{"paths": arr})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) ListVMs(ctx context.Context) ([]VM, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "List"), nil)
	if err != nil {
		return nil, err
	}
	items := getList(resp, "items")
	out := make([]VM, 0, len(items))
	for _, item := range items {
		out = append(out, flattenVM(item))
	}
	return out, nil
}

func (c *grpcClient) GetVM(ctx context.Context, target string) (*VM, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "Get"), map[string]any{"target_id": target})
	if err != nil {
		return nil, err
	}
	vm := flattenVM(resp)
	return &vm, nil
}

func (c *grpcClient) GetVMWithParams(ctx context.Context, target string, withGuestTools bool) (*VM, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "GetWithParams"), map[string]any{
		"target_id":   target,
		"guest_tools": withGuestTools,
	})
	if err != nil {
		return nil, err
	}
	vm := flattenVM(resp)
	return &vm, nil
}

func (c *grpcClient) ValidateVM(ctx context.Context, vm *VM) (*ValidateResult, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "Validate"), encodeVM(vm))
	if err != nil {
		return nil, err
	}
	vr := &ValidateResult{}
	for _, item := range getList(resp, "error_messages") {
		vr.Errors = append(vr.Errors, ValidationError{
			Field:   getString(item, "field"),
			Message: getString(item, "error_message"),
		})
	}
	if pvm, ok := getMap(resp, "Pvm"); ok {
		vmCopy := flattenVM(pvm)
		vr.VM = &vmCopy
		return vr, nil
	}
	if pvm, ok := getMap(resp, "pvm"); ok {
		vmCopy := flattenVM(pvm)
		vr.VM = &vmCopy
	}
	return vr, nil
}

func (c *grpcClient) CreateVM(ctx context.Context, req *CreateVMRequest) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "Create"), map[string]any{
		"start": req.Start,
		"vm":    encodeVM(req.VM),
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) UpdateVM(ctx context.Context, vm *VM) (string, error) {
	payload, err := json.Marshal(vm)
	if err != nil {
		return "", err
	}
	target := firstNonEmpty(vm.DeploymentName, vm.Name)
	if target == "" {
		return "", fmt.Errorf("vm update requires deployment_name or name")
	}
	resp, err := c.unary(ctx, fullMethod("Task", "Create"), map[string]any{
		"method": "vm.update",
		"target": target,
		"args":   base64.StdEncoding.EncodeToString(payload),
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) DeleteVM(ctx context.Context, target string) (string, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "Delete"), map[string]any{"target_id": target})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) MigrateVMDatastore(ctx context.Context, target string, items map[string]VMDatastoreMigrationItem) (string, error) {
	if target == "" {
		return "", fmt.Errorf("vm datastore migration requires vm target")
	}
	if len(items) == 0 {
		return "", fmt.Errorf("vm datastore migration requires at least one item")
	}
	encodedItems := make(map[string]any, len(items))
	for src, item := range items {
		if src == "" {
			return "", fmt.Errorf("vm datastore migration item key cannot be empty")
		}
		if item.ID == "" {
			return "", fmt.Errorf("vm datastore migration item %q requires destination datastore id", src)
		}
		v := map[string]any{"id": item.ID}
		if item.PType > 0 {
			v["ptype"] = int64(item.PType)
		}
		encodedItems[src] = v
	}
	payload, err := json.Marshal(map[string]any{"items": encodedItems})
	if err != nil {
		return "", err
	}
	resp, err := c.unary(ctx, fullMethod("Task", "Create"), map[string]any{
		"method": "vm.migrate_datastore",
		"target": target,
		"args":   base64.StdEncoding.EncodeToString(payload),
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) SetVMPowerState(ctx context.Context, target, desired string, force bool) (string, error) {
	method := ""
	switch strings.ToLower(desired) {
	case "", "unchanged":
		return "", nil
	case "running", "powered_on", "on":
		method = "vm.power_on"
	case "stopped", "powered_off", "off":
		if force {
			method = "vm.hard_stop"
		} else {
			method = "vm.power_off"
		}
	case "suspended":
		method = "vm.suspend"
	default:
		return "", fmt.Errorf("unsupported power_state %q", desired)
	}
	resp, err := c.unary(ctx, fullMethod("Task", "Create"), map[string]any{"method": method, "target": target, "args": ""})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) ListVMSnapshots(ctx context.Context, target string) ([]Snapshot, int64, error) {
	resp, err := c.unary(ctx, fullMethod("Vms", "Snapshots"), map[string]any{"target_id": target})
	if err != nil {
		return nil, 0, err
	}
	items := getList(resp, "items")
	out := make([]Snapshot, 0, len(items))
	for _, item := range items {
		snap := Snapshot{
			ID:            getInt64(item, "id"),
			Name:          getString(item, "name"),
			Description:   getString(item, "description"),
			Timestamp:     getInt64(item, "timestamp"),
			Size:          getInt64(item, "size"),
			QuiesceFS:     getBool(item, "quiesce_fs"),
			VMDescription: getString(item, "vm_description"),
			ParentID:      int32(getInt64(item, "parent_id")),
		}
		if mem, ok := getMap(item, "memory"); ok {
			snap.MemoryEnabled = getBool(mem, "enabled")
			snap.MemorySource = getString(mem, "source")
		}
		for _, d := range getList(item, "disks") {
			snap.Disks = append(snap.Disks, SnapshotDisk{Source: getString(d, "source"), Target: getString(d, "target")})
		}
		out = append(out, snap)
	}
	return out, getInt64(resp, "current_id"), nil
}

func (c *grpcClient) TakeVMSnapshot(ctx context.Context, target, name, description string, includeMemory, quiesceFS bool) (string, error) {
	args, err := json.Marshal(map[string]any{
		"name":           name,
		"description":    description,
		"include_memory": includeMemory,
		"quiesce_fs":     quiesceFS,
	})
	if err != nil {
		return "", err
	}
	resp, err := c.unary(ctx, fullMethod("Task", "Create"), map[string]any{"method": "vm.take_snapshot", "target": target, "args": base64.StdEncoding.EncodeToString(args)})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) DeleteVMSnapshot(ctx context.Context, target string, snapshotID int64) (string, error) {
	args, err := json.Marshal(map[string]any{
		"target":      target,
		"snapshot_id": snapshotID,
	})
	if err != nil {
		return "", err
	}
	resp, err := c.unary(ctx, fullMethod("Task", "Create"), map[string]any{"method": "vm.delete_snapshot", "target": target, "args": base64.StdEncoding.EncodeToString(args)})
	if err != nil {
		return "", err
	}
	return getString(resp, "task_id"), nil
}

func (c *grpcClient) WaitTask(ctx context.Context, taskID string) (*Task, error) {
	if taskID == "" || taskID == "00000000" {
		return &Task{ID: taskID, Status: 2}, nil
	}
	resp, err := c.serverStreamFirst(ctx, fullMethod("Task", "Wait"), map[string]any{"target_id": taskID})
	if err != nil {
		return nil, err
	}
	t := flattenTask(resp)
	if t == nil {
		return nil, fmt.Errorf("task %s returned empty response", taskID)
	}
	switch t.Status {
	case 2:
		return t, nil
	case 3:
		if t.Error != "" {
			return nil, fmt.Errorf("task %s failed: %s", taskID, t.Error)
		}
		return nil, fmt.Errorf("task %s failed", taskID)
	default:
		if t.Error != "" {
			return nil, fmt.Errorf("task %s finished with status %d: %s", taskID, t.Status, t.Error)
		}
		return nil, fmt.Errorf("task %s finished with unexpected status %d", taskID, t.Status)
	}
}

func flattenDatastoreItem(m map[string]any) DatastoreItem {
	item := DatastoreItem{
		Name:            getString(m, "name"),
		Type:            uint32(getInt64(m, "type")),
		Size:            uint64(getInt64(m, "size")),
		ModifiedTime:    getInt64(m, "modified_time"),
		Path:            getString(m, "path"),
		ProvisionedType: uint32(getInt64(m, "provisioned_type")),
	}
	for _, child := range getList(m, "children") {
		item.Children = append(item.Children, flattenDatastoreItem(child))
	}
	return item
}

func flattenDatastore(m map[string]any) Datastore {
	cap := nestedMap(m, "capacity")
	return Datastore{
		ID:               getString(m, "id"),
		Name:             getString(m, "name"),
		PoolName:         getString(m, "pool_name"),
		TypeCode:         int32(getInt64(m, "type_code")),
		State:            uint32(getInt64(m, "state")),
		Status:           uint32(getInt64(m, "status")),
		DriveType:        getString(m, "drive_type"),
		CapacityMB:       getFloat64(cap, "capacity_mb"),
		ProvisionedMB:    getFloat64(cap, "provisioned_mb"),
		FreeMB:           getFloat64(cap, "free_mb"),
		UsedMB:           getFloat64(cap, "used_mb"),
		ThinProvisioning: getBool(m, "thin_provisioning"),
		AccessMode:       getString(m, "access_mode"),
		Server:           nestedString(m, "connectivity", "endpoint"),
	}
}

func flattenVM(m map[string]any) VM {
	vm := VM{
		DeploymentName: getString(m, "deployment_name"),
		Name:           getString(m, "name"),
		UUID:           getString(m, "uuid"),
		Compatibility:  getString(m, "compatibility"),
		GuestOSFamily:  getString(m, "guest_os_family"),
		GuestOSVersion: getString(m, "guest_os_version"),
		MachineType:    getString(m, "machine_type"),
		Storage: VMStorage{
			ID:     nestedString(m, "storage", "id"),
			Folder: nestedString(m, "storage", "folder"),
		},
		CPU: VMCPU{
			VCPUs:          int32(nestedInt64(m, "cpu", "vcpus")),
			MaxVCPUs:       int32(nestedInt64(m, "cpu", "max_vcpus")),
			CorePerSocket:  int32(nestedInt64(m, "cpu", "core_per_socket")),
			Model:          nestedString(m, "cpu", "model"),
			ReservationMHz: int32(nestedInt64(m, "cpu", "reservation_mhz")),
			LimitMHz:       int32(nestedInt64(m, "cpu", "limit_mhz")),
			Shares:         int32(nestedInt64(m, "cpu", "shares")),
			Hotplug:        nestedBool(m, "cpu", "hotplug"),
		},
		Memory: VMMemory{
			SizeMB:        int32(nestedInt64(m, "memory", "size_mb")),
			Hotplug:       nestedBool(m, "memory", "hotplug"),
			ReservationMB: int32(nestedInt64(m, "memory", "reservation_mb")),
			LimitMB:       int32(nestedInt64(m, "memory", "limit_mb")),
		},
		VideoCard: VMVideo{
			Adapter:  nestedString(m, "video_card", "adapter"),
			Displays: int32(nestedInt64(m, "video_card", "displays")),
			MemoryMB: int32(nestedInt64(m, "video_card", "memory_mb")),
		},
		IsTemplate:       getBool(m, "IsTemplate") || getBool(m, "is_template"),
		MonitoringState:  uint32(nestedInt64(m, "monitoring", "state")),
		MonitoringStatus: uint32(nestedInt64(m, "monitoring", "status")),
	}
	if mon, ok := getMap(m, "monitoring"); ok {
		if gt, ok := getMap(mon, "guest_tools"); ok {
			vm.GuestToolsInfo = VMGuestToolsInfo{
				Status:  getString(gt, "status"),
				Version: getString(gt, "version"),
				IP:      getString(gt, "ip"),
				DNSName: getString(gt, "dns_name"),
			}
		}
	}
	for _, item := range getList(m, "usb_controllers") {
		vm.USBControllers = append(vm.USBControllers, USBController{Type: getString(item, "type")})
	}
	for _, item := range getList(m, "input_devices") {
		vm.InputDevices = append(vm.InputDevices, InputDevice{Type: getString(item, "type"), Bus: getString(item, "bus")})
	}
	if opts, ok := getMap(m, "options"); ok {
		vm.Options.RemoteConsole = RemoteConsole{
			Type:          nestedString(opts, "remote_console", "type"),
			Port:          int32(nestedInt64(opts, "remote_console", "port")),
			Keymap:        nestedString(opts, "remote_console", "keymap"),
			Password:      nestedString(opts, "remote_console", "password"),
			GuestOSLock:   nestedBool(opts, "remote_console", "guest_os_lock"),
			LimitSessions: int32(nestedInt64(opts, "remote_console", "limit_sessions")),
			Spice: Spice{
				ImgCompression:      nestedString(nestedMap(opts, "remote_console"), "spice", "img_compression"),
				JpegCompression:     nestedString(nestedMap(opts, "remote_console"), "spice", "jpeg_compression"),
				ZlibGlzCompression:  nestedString(nestedMap(opts, "remote_console"), "spice", "zlib_glz_compression"),
				StreamingMode:       nestedString(nestedMap(opts, "remote_console"), "spice", "streaming_mode"),
				PlaybackCompression: nestedBool(nestedMap(opts, "remote_console"), "spice", "playback_compression"),
				FileTransfer:        nestedBool(nestedMap(opts, "remote_console"), "spice", "file_transfer"),
				Clipboard:           nestedBool(nestedMap(opts, "remote_console"), "spice", "clipboard"),
			},
		}
		vm.Options.GuestTools = GuestTools{
			Enabled:          nestedBool(opts, "guest_tools", "enabled"),
			SynchronizedTime: nestedBool(opts, "guest_tools", "synchronized_time"),
			Balloon:          nestedBool(opts, "guest_tools", "balloon"),
		}
		vm.Options.BootOptions = BootOptions{
			Firmware:    nestedString(opts, "boot_options", "firmware"),
			BootDelayMS: int32(nestedInt64(opts, "boot_options", "boot_delay_ms")),
			BootMenu:    nestedBool(opts, "boot_options", "boot_menu"),
		}
	}
	for _, item := range getList(m, "disk_devices") {
		vm.DiskDevices = append(vm.DiskDevices, DiskDevice{
			Size:          uint64(getInt64(item, "size")),
			Source:        getString(item, "source"),
			StorageID:     getString(item, "storage_id"),
			DeviceType:    getString(item, "device_type"),
			Bus:           getString(item, "bus"),
			Target:        getString(item, "target"),
			BootOrder:     int32(getInt64(item, "boot_order")),
			ProvisionType: getString(item, "provision_type"),
			DiskMode:      getString(item, "disk_mode"),
			ReadOnly:      getBool(item, "read_only"),
			Create:        getBool(item, "create"),
			Remove:        getBool(item, "remove"),
			Attach:        getBool(item, "attach"),
			Detach:        getBool(item, "detach"),
			Resize:        getBool(item, "resize"),
		})
	}
	for _, item := range getList(m, "network_devices") {
		vm.NetworkDevices = append(vm.NetworkDevices, NetworkDevice{
			Network:   getString(item, "network"),
			NetBridge: getString(item, "net_bridge"),
			MAC:       getString(item, "mac"),
			Target:    getString(item, "target"),
			Model:     getString(item, "model"),
			BootOrder: int32(getInt64(item, "boot_order")),
			VLAN:      int32(getInt64(item, "vlan")),
		})
	}
	vm.PowerState = vmPowerState(vm.MonitoringState)
	return vm
}

func vmPowerState(state uint32) string {
	switch state {
	case 2:
		return "running"
	case 1:
		return "stopped"
	case 3:
		return "suspended"
	case 7:
		return "paused"
	case 9, 10:
		return "stopping"
	case 4, 12:
		return "error"
	case 6:
		return "pending"
	default:
		return "unknown"
	}
}

func flattenTask(m map[string]any) *Task {
	t := &Task{
		ID:          getString(m, "id"),
		Method:      getString(m, "method"),
		Status:      uint32(getInt64(m, "status")),
		Error:       getString(m, "error"),
		Target:      getString(m, "target"),
		CreatedID:   nestedString(m, "extra", "created_id"),
		CreatedName: nestedString(m, "extra", "created_name"),
	}
	if v := getInt64(m, "completion"); v > 0 {
		tm := time.Unix(v, 0)
		t.CompletedAt = &tm
	}
	return t
}

func encodeVM(vm *VM) map[string]any {
	if vm == nil {
		return map[string]any{}
	}
	res := map[string]any{
		"deployment_name":  vm.DeploymentName,
		"name":             vm.Name,
		"uuid":             vm.UUID,
		"compatibility":    vm.Compatibility,
		"guest_os_family":  vm.GuestOSFamily,
		"guest_os_version": vm.GuestOSVersion,
		"machine_type":     vm.MachineType,
		"storage": map[string]any{
			"id":     vm.Storage.ID,
			"folder": vm.Storage.Folder,
		},
		"cpu": map[string]any{
			"vcpus":           vm.CPU.VCPUs,
			"max_vcpus":       vm.CPU.MaxVCPUs,
			"core_per_socket": vm.CPU.CorePerSocket,
			"model":           vm.CPU.Model,
			"reservation_mhz": vm.CPU.ReservationMHz,
			"limit_mhz":       vm.CPU.LimitMHz,
			"shares":          vm.CPU.Shares,
			"hotplug":         vm.CPU.Hotplug,
		},
		"memory": map[string]any{
			"size_mb":        vm.Memory.SizeMB,
			"hotplug":        vm.Memory.Hotplug,
			"reservation_mb": vm.Memory.ReservationMB,
			"limit_mb":       vm.Memory.LimitMB,
		},
		"video_card": map[string]any{
			"adapter":   vm.VideoCard.Adapter,
			"displays":  vm.VideoCard.Displays,
			"memory_mb": vm.VideoCard.MemoryMB,
		},
		"options": map[string]any{
			"remote_console": map[string]any{
				"type":           vm.Options.RemoteConsole.Type,
				"port":           vm.Options.RemoteConsole.Port,
				"keymap":         vm.Options.RemoteConsole.Keymap,
				"password":       vm.Options.RemoteConsole.Password,
				"guest_os_lock":  vm.Options.RemoteConsole.GuestOSLock,
				"limit_sessions": vm.Options.RemoteConsole.LimitSessions,
				"spice": map[string]any{
					"img_compression":      vm.Options.RemoteConsole.Spice.ImgCompression,
					"jpeg_compression":     vm.Options.RemoteConsole.Spice.JpegCompression,
					"zlib_glz_compression": vm.Options.RemoteConsole.Spice.ZlibGlzCompression,
					"streaming_mode":       vm.Options.RemoteConsole.Spice.StreamingMode,
					"playback_compression": vm.Options.RemoteConsole.Spice.PlaybackCompression,
					"file_transfer":        vm.Options.RemoteConsole.Spice.FileTransfer,
					"clipboard":            vm.Options.RemoteConsole.Spice.Clipboard,
				},
			},
			"guest_tools": map[string]any{
				"enabled":           vm.Options.GuestTools.Enabled,
				"synchronized_time": vm.Options.GuestTools.SynchronizedTime,
				"balloon":           vm.Options.GuestTools.Balloon,
			},
			"boot_options": map[string]any{
				"firmware":      vm.Options.BootOptions.Firmware,
				"boot_delay_ms": vm.Options.BootOptions.BootDelayMS,
				"boot_menu":     vm.Options.BootOptions.BootMenu,
			},
		},
		"IsTemplate": vm.IsTemplate,
	}
	if len(vm.USBControllers) > 0 {
		items := make([]any, 0, len(vm.USBControllers))
		for _, u := range vm.USBControllers {
			items = append(items, map[string]any{"type": u.Type})
		}
		res["usb_controllers"] = items
	}
	if len(vm.InputDevices) > 0 {
		items := make([]any, 0, len(vm.InputDevices))
		for _, d := range vm.InputDevices {
			items = append(items, map[string]any{"type": d.Type, "bus": d.Bus})
		}
		res["input_devices"] = items
	}
	if len(vm.DiskDevices) > 0 {
		disks := make([]any, 0, len(vm.DiskDevices))
		for _, d := range vm.DiskDevices {
			disks = append(disks, map[string]any{
				"size":           d.Size,
				"source":         d.Source,
				"storage_id":     d.StorageID,
				"device_type":    d.DeviceType,
				"bus":            d.Bus,
				"target":         d.Target,
				"boot_order":     d.BootOrder,
				"provision_type": d.ProvisionType,
				"disk_mode":      d.DiskMode,
				"read_only":      d.ReadOnly,
				"create":         d.Create,
				"remove":         d.Remove,
				"attach":         d.Attach,
				"detach":         d.Detach,
				"resize":         d.Resize,
			})
		}
		res["disk_devices"] = disks
	}
	if len(vm.NetworkDevices) > 0 {
		nics := make([]any, 0, len(vm.NetworkDevices))
		for _, n := range vm.NetworkDevices {
			nics = append(nics, map[string]any{
				"network":    n.Network,
				"net_bridge": n.NetBridge,
				"mac":        n.MAC,
				"target":     n.Target,
				"model":      n.Model,
				"boot_order": n.BootOrder,
				"vlan":       n.VLAN,
			})
		}
		res["network_devices"] = nics
	}
	return res
}

func fullMethod(service, method string) string {
	return fmt.Sprintf("/%s.%s/%s", corePackage, service, method)
}

func (c *grpcClient) unary(ctx context.Context, method string, reqFields map[string]any) (map[string]any, error) {
	reqDesc, respDesc, err := c.methodDescriptors(method)
	if err != nil {
		return nil, err
	}
	req := dynamicpb.NewMessage(reqDesc)
	if err := setFields(req, reqFields); err != nil {
		return nil, err
	}
	resp := dynamicpb.NewMessage(respDesc)
	if err := c.conn.Invoke(c.withAuth(ctx), method, req, resp); err != nil {
		return nil, normalizeError(err)
	}
	return messageToMap(resp), nil
}

func (c *grpcClient) serverStreamAll(ctx context.Context, method string, reqFields map[string]any) ([]map[string]any, error) {
	reqDesc, respDesc, err := c.methodDescriptors(method)
	if err != nil {
		return nil, err
	}
	req := dynamicpb.NewMessage(reqDesc)
	if err := setFields(req, reqFields); err != nil {
		return nil, err
	}
	streamDesc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: false}
	stream, err := c.conn.NewStream(c.withAuth(ctx), streamDesc, method)
	if err != nil {
		return nil, normalizeError(err)
	}
	if err := stream.SendMsg(req); err != nil {
		return nil, normalizeError(err)
	}
	if err := stream.CloseSend(); err != nil {
		return nil, normalizeError(err)
	}
	out := []map[string]any{}
	for {
		resp := dynamicpb.NewMessage(respDesc)
		if err := stream.RecvMsg(resp); err != nil {
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.OutOfRange {
				break
			}
			return nil, normalizeError(err)
		}
		out = append(out, messageToMap(resp))
	}
	return out, nil
}

func (c *grpcClient) serverStreamFirst(ctx context.Context, method string, reqFields map[string]any) (map[string]any, error) {
	reqDesc, respDesc, err := c.methodDescriptors(method)
	if err != nil {
		return nil, err
	}
	req := dynamicpb.NewMessage(reqDesc)
	if err := setFields(req, reqFields); err != nil {
		return nil, err
	}
	streamDesc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: false}
	stream, err := c.conn.NewStream(c.withAuth(ctx), streamDesc, method)
	if err != nil {
		return nil, normalizeError(err)
	}
	if err := stream.SendMsg(req); err != nil {
		return nil, normalizeError(err)
	}
	if err := stream.CloseSend(); err != nil {
		return nil, normalizeError(err)
	}
	resp := dynamicpb.NewMessage(respDesc)
	if err := stream.RecvMsg(resp); err != nil {
		return nil, normalizeError(err)
	}
	return messageToMap(resp), nil
}

func (c *grpcClient) withAuth(ctx context.Context) context.Context {
	if c.cfg.Token != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", bearer(c.cfg.Token))
	}
	return ctx
}

func normalizeError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	if st.Code() == codes.NotFound {
		return ErrNotFound
	}
	return err
}

func (c *grpcClient) methodDescriptors(method string) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	trimmed := strings.TrimPrefix(method, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid method path %q", method)
	}
	serviceName, methodName := parts[0], parts[1]
	if err := c.ensureSymbol(serviceName); err != nil {
		return nil, nil, err
	}
	d, err := c.files.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, nil, err
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, nil, fmt.Errorf("descriptor %q is not service", serviceName)
	}
	md := svc.Methods().ByName(protoreflect.Name(methodName))
	if md == nil {
		return nil, nil, fmt.Errorf("method %s not found in %s", methodName, serviceName)
	}
	return md.Input(), md.Output(), nil
}

func (c *grpcClient) ensureSymbol(symbol string) error {
	c.filesMu.Lock()
	defer c.filesMu.Unlock()
	if _, err := c.files.FindDescriptorByName(protoreflect.FullName(symbol)); err == nil {
		return nil
	}
	stream, err := grpc_reflection_v1alpha.NewServerReflectionClient(c.conn).ServerReflectionInfo(c.withAuth(context.Background()))
	if err != nil {
		return err
	}
	defer stream.CloseSend()
	if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}); err != nil {
		return err
	}
	resp, err := stream.Recv()
	if err != nil {
		return err
	}
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		if er := resp.GetErrorResponse(); er != nil {
			return fmt.Errorf("reflection error: %s", er.ErrorMessage)
		}
		return fmt.Errorf("reflection returned no file descriptor for %s", symbol)
	}
	pending := make([]*descriptorpb.FileDescriptorProto, 0, len(fdResp.FileDescriptorProto))
	for _, raw := range fdResp.FileDescriptorProto {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(raw, fd); err != nil {
			return err
		}
		pending = append(pending, fd)
	}
	for len(pending) > 0 {
		progress := false
		next := pending[:0]
		for _, fd := range pending {
			file, err := protodesc.NewFile(fd, c.files)
			if err != nil {
				next = append(next, fd)
				continue
			}
			if err := c.files.RegisterFile(file); err == nil {
				progress = true
			}
		}
		if !progress {
			break
		}
		pending = next
	}
	if _, err := c.files.FindDescriptorByName(protoreflect.FullName(symbol)); err != nil {
		return err
	}
	return nil
}

func setFields(msg *dynamicpb.Message, values map[string]any) error {
	if values == nil {
		return nil
	}
	for k, v := range values {
		if v == nil {
			continue
		}
		fd := fieldByJSONOrTextName(msg.Descriptor(), k)
		if fd == nil {
			continue
		}
		val, err := toValue(fd, v)
		if err != nil {
			return fmt.Errorf("set field %s: %w", k, err)
		}
		msg.Set(fd, val)
	}
	return nil
}

func fieldByJSONOrTextName(md protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		if string(f.Name()) == name || f.JSONName() == name || strings.EqualFold(string(f.Name()), name) || strings.EqualFold(f.JSONName(), name) {
			return f
		}
	}
	return nil
}

func toValue(fd protoreflect.FieldDescriptor, v any) (protoreflect.Value, error) {
	if fd.IsList() {
		list := dynamicpb.NewMessage(fd.ContainingMessage()).NewField(fd).List()
		switch vv := v.(type) {
		case []any:
			for _, item := range vv {
				pv, err := toSingularValue(fd, item)
				if err != nil {
					return protoreflect.Value{}, err
				}
				list.Append(pv)
			}
		case []map[string]any:
			for _, item := range vv {
				pv, err := toSingularValue(fd, item)
				if err != nil {
					return protoreflect.Value{}, err
				}
				list.Append(pv)
			}
		default:
			return protoreflect.Value{}, fmt.Errorf("unsupported list type %T", v)
		}
		return protoreflect.ValueOfList(list), nil
	}
	return toSingularValue(fd, v)
}

func toSingularValue(fd protoreflect.FieldDescriptor, v any) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(fmt.Sprint(v)), nil
	case protoreflect.BoolKind:
		if b, ok := v.(bool); ok {
			return protoreflect.ValueOfBool(b), nil
		}
		return protoreflect.ValueOfBool(fmt.Sprint(v) == "true"), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(asInt64(v))), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(asInt64(v)), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(asInt64(v))), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(uint64(asInt64(v))), nil
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return protoreflect.ValueOfFloat64(asFloat64(v)), nil
	case protoreflect.BytesKind:
		s, ok := v.(string)
		if ok {
			return protoreflect.ValueOfBytes([]byte(s)), nil
		}
		return protoreflect.ValueOfBytes(v.([]byte)), nil
	case protoreflect.MessageKind:
		m := dynamicpb.NewMessage(fd.Message())
		mv, ok := v.(map[string]any)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("expected map for message %s, got %T", fd.FullName(), v)
		}
		if err := setFields(m, mv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfMessage(m), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind %s", fd.Kind())
	}
}

func messageToMap(msg *dynamicpb.Message) map[string]any {
	out := map[string]any{}
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		key := string(fd.Name())
		if key == "Pvm" {
			key = "Pvm"
		}
		out[key] = valueToAny(fd, v)
		return true
	})
	return out
}

func valueToAny(fd protoreflect.FieldDescriptor, v protoreflect.Value) any {
	if fd.IsList() {
		list := v.List()
		items := make([]any, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			items = append(items, singularValueToAny(fd.Kind(), list.Get(i)))
		}
		return items
	}
	return singularValueToAny(fd.Kind(), v)
}

func singularValueToAny(kind protoreflect.Kind, v protoreflect.Value) any {
	switch kind {
	case protoreflect.StringKind:
		return v.String()
	case protoreflect.BoolKind:
		return v.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return v.Int()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return int64(v.Uint())
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return v.Float()
	case protoreflect.BytesKind:
		return append([]byte(nil), v.Bytes()...)
	case protoreflect.MessageKind:
		if m, ok := v.Message().Interface().(*dynamicpb.Message); ok {
			return messageToMap(m)
		}
		return nil
	default:
		return v.Interface()
	}
}

func getMap(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	mv, ok := v.(map[string]any)
	return mv, ok
}

func nestedMap(m map[string]any, key string) map[string]any {
	mv, _ := getMap(m, key)
	return mv
}

func getList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	res := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mv, ok := item.(map[string]any); ok {
			res = append(res, mv)
		}
	}
	return res
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		switch vv := v.(type) {
		case string:
			return vv
		case []byte:
			return string(vv)
		}
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func getInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	return asInt64(m[key])
}

func getFloat64(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	return asFloat64(m[key])
}

func nestedString(m map[string]any, key, child string) string {
	return getString(nestedMap(m, key), child)
}

func nestedInt64(m map[string]any, key, child string) int64 {
	return getInt64(nestedMap(m, key), child)
}

func nestedBool(m map[string]any, key, child string) bool {
	return getBool(nestedMap(m, key), child)
}

func asInt64(v any) int64 {
	switch vv := v.(type) {
	case int:
		return int64(vv)
	case int32:
		return int64(vv)
	case int64:
		return vv
	case uint32:
		return int64(vv)
	case uint64:
		return int64(vv)
	case float32:
		return int64(vv)
	case float64:
		return int64(vv)
	default:
		return 0
	}
}

func asFloat64(v any) float64 {
	switch vv := v.(type) {
	case int:
		return float64(vv)
	case int32:
		return float64(vv)
	case int64:
		return float64(vv)
	case uint32:
		return float64(vv)
	case uint64:
		return float64(vv)
	case float32:
		return float64(vv)
	case float64:
		return vv
	default:
		return 0
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
