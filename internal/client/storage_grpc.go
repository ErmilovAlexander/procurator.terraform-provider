package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

const storagePackage = "procurator.storage.api"

func NewStorage(cfg Config) (StorageClient, error) {
	endpoint := cfg.StorageEndpoint
	if endpoint == "" {
		endpoint = cfg.Endpoint
	}
	if endpoint == "" {
		return nil, fmt.Errorf("storage endpoint is required")
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
	conn, err := grpc.Dial(endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}
	c := &grpcClient{cfg: cfg, conn: conn, files: new(protoregistry.Files), messages: map[string]protoreflect.MessageDescriptor{}, filesMu: sync.Mutex{}}
	return c, nil
}

func fullMethodStorage(service, method string) string {
	return fmt.Sprintf("/%s.%s/%s", storagePackage, service, method)
}

func (c *grpcClient) ListStorageAdapters(ctx context.Context) ([]StorageAdapter, error) {
	resp, err := c.unary(ctx, fullMethodStorage("Storage", "GetStorageAdapters"), nil)
	if err != nil {
		return nil, err
	}
	items := getList(resp, "items")
	out := make([]StorageAdapter, 0, len(items))
	for _, item := range items {
		out = append(out, flattenStorageAdapter(item))
	}
	return out, nil
}

func (c *grpcClient) ListStorageDevices(ctx context.Context) ([]StorageDevice, error) {
	resp, err := c.unary(ctx, fullMethodStorage("Storage", "GetStorageDevices"), nil)
	if err != nil {
		return nil, err
	}
	items := getList(resp, "items")
	out := make([]StorageDevice, 0, len(items))
	for _, item := range items {
		out = append(out, flattenStorageDevice(item))
	}
	return out, nil
}

func flattenStorageAdapter(m map[string]any) StorageAdapter {
	status := nestedMap(m, "status")
	rescan := nestedMap(m, "rescanStorage")
	return StorageAdapter{
		Adapter:              getString(m, "adapter"),
		Devices:              int32(getInt64(m, "devices")),
		ID:                   firstNonEmpty(getString(m, "id"), getString(m, "Id")),
		Identifier:           getString(m, "identifier"),
		Model:                getString(m, "model"),
		Path:                 int32(getInt64(m, "path")),
		StatusText:           firstNonEmpty(getString(status, "text"), getString(status, "Text")),
		StatusValue:          firstNonEmpty(getString(status, "value"), getString(status, "Value")),
		Targets:              int32(getInt64(m, "targets")),
		Type:                 getString(m, "type"),
		RescanStorageAdapter: getBool(m, "rescanStorageAdapter"),
		ScanStorageDevice:    firstBool(rescan, "scanStorageDevice", "ScanStorageDevice"),
		ScanDatastore:        firstBool(rescan, "scanDatastore", "ScanDatastore"),
		ProtectDatastore:     firstBool(rescan, "protectDatastore", "ProtectDatastore"),
		Response:             firstNonEmpty(getString(m, "response"), getString(m, "Response")),
	}
}

func flattenStorageDevice(m map[string]any) StorageDevice {
	ds := nestedMap(m, "datastore")
	return StorageDevice{
		Adapter:              getString(m, "adapter"),
		CapacityMB:           int32(getInt64(m, "capacity_mb")),
		DatastoreType:        getString(ds, "type"),
		DatastoreID:          getString(ds, "id"),
		DatastoreName:        getString(ds, "name"),
		DriveType:            getString(m, "drive_type"),
		HardwareAcceleration: getString(m, "hardware_acceleration"),
		ID:                   firstNonEmpty(getString(m, "id"), getString(m, "Id")),
		Identifier:           getString(m, "identifier"),
		LUN:                  int32(getInt64(m, "lun")),
		Name:                 getString(m, "name"),
		OperationalState:     getString(m, "operational_state"),
		Owner:                getString(m, "owner"),
		PerenniallyReserved:  getString(m, "perennially_reserved"),
		PhysicalLocation:     getString(m, "physical_location"),
		SectorFormat:         getString(m, "sector_format"),
		Transport:            getString(m, "transport"),
		Type:                 getString(m, "type"),
		StorageInterface:     normalizeStorageInterface(getInt64(m, "storageInterface")),
	}
}

func firstBool(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return getBool(m, k)
		}
	}
	return false
}

func normalizeStorageInterface(v int64) string {
	switch v {
	case 1:
		return "sas"
	case 2:
		return "sata"
	case 3:
		return "nvme"
	case 4:
		return "scsi_unknown"
	case 5:
		return "san_iscsi"
	case 6:
		return "san_fc"
	default:
		return "unknown"
	}
}
