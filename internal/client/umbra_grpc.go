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

const umbraPackage = "umbra"

func NewUmbra(cfg Config) (UmbraClient, error) {
	endpoint := cfg.UmbraEndpoint
	if endpoint == "" {
		endpoint = cfg.Endpoint
	}
	if endpoint == "" {
		return nil, fmt.Errorf("umbra endpoint is required")
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

func fullMethodUmbra(service, method string) string {
	return fmt.Sprintf("/%s.%s/%s", umbraPackage, service, method)
}

func (c *grpcClient) ListNetworks(ctx context.Context, filter string) ([]Network, error) {
	resp, err := c.serverStreamAll(ctx, fullMethodUmbra("Networks", "List"), map[string]any{"filter": filter})
	if err != nil {
		return nil, err
	}
	out := make([]Network, 0, len(resp))
	for _, item := range resp {
		out = append(out, flattenNetwork(item))
	}
	return out, nil
}

func (c *grpcClient) GetNetworkByID(ctx context.Context, id string) (*Network, error) {
	if id == "" {
		return nil, fmt.Errorf("network id is required")
	}
	resp, err := c.unary(ctx, fullMethodUmbra("Networks", "Info"), map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	n := flattenNetwork(resp)
	return &n, nil
}

func (c *grpcClient) GetNetworkByName(ctx context.Context, name string) (*Network, error) {
	if name == "" {
		return nil, fmt.Errorf("network name is required")
	}
	resp, err := c.unary(ctx, fullMethodUmbra("Networks", "InfoByName"), map[string]any{"name": name})
	if err != nil {
		return nil, err
	}
	n := flattenNetwork(resp)
	return &n, nil
}

func (c *grpcClient) CreateNetwork(ctx context.Context, req *NetworkCreateRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("network create request is nil")
	}
	if req.Name == "" {
		return "", fmt.Errorf("network name is required")
	}
	if req.SwitchID == "" {
		return "", fmt.Errorf("switch_id is required")
	}
	resp, err := c.unary(ctx, fullMethodUmbra("Networks", "Create"), map[string]any{
		"name":   req.Name,
		"vlan":   req.VLAN,
		"switch": req.SwitchID,
	})
	if err != nil {
		return "", err
	}
	return getString(resp, "id"), nil
}

func (c *grpcClient) UpdateNetwork(ctx context.Context, n *Network) error {
	if n == nil {
		return fmt.Errorf("network is nil")
	}
	if n.ID == "" {
		return fmt.Errorf("network id is required")
	}
	_, err := c.unary(ctx, fullMethodUmbra("Networks", "Update"), map[string]any{
		"id":     n.ID,
		"name":   n.Name,
		"vlan":   n.VLAN,
		"switch": n.SwitchID,
	})
	return err
}

func (c *grpcClient) DeleteNetwork(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("network id is required")
	}
	_, err := c.unary(ctx, fullMethodUmbra("Networks", "Delete"), map[string]any{"id": id})
	return err
}

func (c *grpcClient) ListNICs(ctx context.Context, filter string) ([]NIC, error) {
	resp, err := c.serverStreamAll(ctx, fullMethodUmbra("Nics", "List"), map[string]any{"filter": filter})
	if err != nil {
		return nil, err
	}
	out := make([]NIC, 0, len(resp))
	for _, item := range resp {
		out = append(out, flattenNIC(item))
	}
	return out, nil
}

func (c *grpcClient) ListSwitches(ctx context.Context) ([]Switch, error) {
	resp, err := c.serverStreamAll(ctx, fullMethodUmbra("Switches", "List"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]Switch, 0, len(resp))
	for _, item := range resp {
		out = append(out, flattenSwitch(item))
	}
	return out, nil
}

func (c *grpcClient) GetSwitch(ctx context.Context, id string) (*Switch, error) {
	if id == "" {
		return nil, fmt.Errorf("switch id is required")
	}
	resp, err := c.unary(ctx, fullMethodUmbra("Switches", "Info"), map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	sw := flattenSwitch(resp)
	return &sw, nil
}

func (c *grpcClient) CreateSwitch(ctx context.Context, req *SwitchCreateRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("switch create request is nil")
	}
	payload := map[string]any{
		"mtu": req.MTU,
	}
	if hasNICs(req.NICs) {
		payload["nics"] = encodeNICs(req.NICs)
	}
	resp, err := c.unary(ctx, fullMethodUmbra("Switches", "Create"), payload)
	if err != nil {
		return "", err
	}
	return getString(resp, "id"), nil
}

func (c *grpcClient) UpdateSwitch(ctx context.Context, sw *Switch) error {
	if sw == nil {
		return fmt.Errorf("switch is nil")
	}
	if sw.ID == "" {
		return fmt.Errorf("switch id is required")
	}
	payload := map[string]any{
		"id":  sw.ID,
		"mtu": sw.MTU,
	}
	if hasNICs(sw.NICs) {
		payload["nics"] = encodeNICs(sw.NICs)
	}
	_, err := c.unary(ctx, fullMethodUmbra("Switches", "Update"), payload)
	return err
}

func (c *grpcClient) DeleteSwitch(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("switch id is required")
	}
	_, err := c.unary(ctx, fullMethodUmbra("Switches", "Delete"), map[string]any{"id": id})
	return err
}

func hasNICs(n NICs) bool {
	return len(n.Active) > 0 || len(n.Standby) > 0 || len(n.Unused) > 0 || len(n.Connected) > 0 || n.Inherit
}

func encodeNICs(n NICs) map[string]any {
	return map[string]any{
		"active":    stringsToAny(n.Active),
		"standby":   stringsToAny(n.Standby),
		"unused":    stringsToAny(n.Unused),
		"connected": stringsToAny(n.Connected),
		"inherit":   n.Inherit,
	}
}

func stringsToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func flattenState(in map[string]any) (string, []string) {
	if in == nil {
		return "", nil
	}
	state := umbraStateName(getInt64(in, "state"))
	var errs []string
	if arr, ok := in["errors"].([]any); ok {
		for _, e := range arr {
			errs = append(errs, fmt.Sprint(e))
		}
	}
	return state, errs
}

func flattenNICs(in map[string]any) NICs {
	return NICs{
		Active:    toStringSlice(in["active"]),
		Standby:   toStringSlice(in["standby"]),
		Unused:    toStringSlice(in["unused"]),
		Connected: toStringSlice(in["connected"]),
		Inherit:   getBool(in, "inherit"),
	}
}

func flattenNIC(in map[string]any) NIC {
	state, errs := flattenState(nestedMap(in, "state"))
	return NIC{
		ID:       getString(in, "id"),
		Adapter:  getString(in, "adapter"),
		PCIAddr:  getString(in, "pci_addr"),
		Driver:   getString(in, "driver"),
		Carrier:  getBool(in, "carrier"),
		Speed:    uint64(getInt64(in, "speed")),
		Duplex:   getString(in, "duplex"),
		Networks: toStringSlice(in["networks"]),
		Sriov:    getBool(in, "sr_iov"),
		CDP:      getBool(in, "cdp"),
		LLDP:     getBool(in, "lldp"),
		Managed:  getBool(in, "managed"),
		Name:     getString(in, "name"),
		SwitchID: getString(in, "switch"),
		MAC:      getString(in, "mac"),
		State:    state,
		Errors:   errs,
	}
}

func flattenSwitch(in map[string]any) Switch {
	state, errs := flattenState(nestedMap(in, "state"))
	return Switch{
		ID:       getString(in, "id"),
		MTU:      uint32(getInt64(in, "mtu")),
		NICs:     flattenNICs(nestedMap(in, "nics")),
		Networks: toStringSlice(in["networks"]),
		State:    state,
		Errors:   errs,
	}
}

func flattenNetwork(in map[string]any) Network {
	state, errs := flattenState(nestedMap(in, "state"))
	return Network{
		ID:          getString(in, "id"),
		Name:        getString(in, "name"),
		VLAN:        int32(getInt64(in, "vlan")),
		NetBridge:   getString(in, "net_bridge"),
		Kind:        umbraNetworkKindName(getInt64(in, "kind")),
		SwitchID:    getString(in, "switch"),
		VmsCount:    uint64(getInt64(in, "vms_count")),
		ActivePorts: uint64(getInt64(in, "active_ports")),
		State:       state,
		Errors:      errs,
	}
}

func umbraStateName(v int64) string {
	switch v {
	case 1:
		return "ok"
	case 2:
		return "warning"
	case 3:
		return "error"
	default:
		return "unknown"
	}
}

func umbraNetworkKindName(v int64) string {
	switch v {
	case 1:
		return "normal"
	case 2:
		return "system"
	default:
		return "unknown"
	}
}

func toStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprint(item))
	}
	return out
}
