package client

import "time"

type Config struct {
	Endpoint        string
	UmbraEndpoint   string
	StorageEndpoint string
	Token           string
	Username        string
	Password        string
	Insecure        bool
	CAFile          string
	Authority       string
}

type StorageAdapter struct {
	Adapter              string
	Devices              int32
	ID                   string
	Identifier           string
	Model                string
	Path                 int32
	StatusText           string
	StatusValue          string
	Targets              int32
	Type                 string
	RescanStorageAdapter bool
	ScanStorageDevice    bool
	ScanDatastore        bool
	ProtectDatastore     bool
	Response             string
}

type StorageDevice struct {
	Adapter              string
	CapacityMB           int32
	DatastoreType        string
	DatastoreID          string
	DatastoreName        string
	DriveType            string
	HardwareAcceleration string
	ID                   string
	Identifier           string
	LUN                  int32
	Name                 string
	OperationalState     string
	Owner                string
	PerenniallyReserved  string
	PhysicalLocation     string
	SectorFormat         string
	Transport            string
	Type                 string
	StorageInterface     string
}

type Datastore struct {
	Server           string
	Folder           string
	Readonly         bool
	Devices          []string
	Reinit           *bool
	NConnect         *int32
	ID               string
	Name             string
	PoolName         string
	TypeCode         int32
	State            uint32
	Status           uint32
	DriveType        string
	CapacityMB       float64
	ProvisionedMB    float64
	FreeMB           float64
	UsedMB           float64
	ThinProvisioning bool
	AccessMode       string
}

type DatastoreItem struct {
	Name            string
	Type            uint32
	Size            uint64
	ModifiedTime    int64
	Path            string
	ProvisionedType uint32
	Children        []DatastoreItem
}

type VMDatastoreMigrationItem struct {
	ID    string
	PType uint32
}

type NICs struct {
	Active    []string
	Standby   []string
	Unused    []string
	Connected []string
	Inherit   bool
}

type NIC struct {
	ID       string
	Adapter  string
	PCIAddr  string
	Driver   string
	Carrier  bool
	Speed    uint64
	Duplex   string
	Networks []string
	Sriov    bool
	CDP      bool
	LLDP     bool
	Managed  bool
	Name     string
	SwitchID string
	MAC      string
	State    string
	Errors   []string
}

type Switch struct {
	ID       string
	MTU      uint32
	NICs     NICs
	Networks []string
	State    string
	Errors   []string
}

type SwitchCreateRequest struct {
	MTU  uint32
	NICs NICs
}

type Network struct {
	ID          string
	Name        string
	VLAN        int32
	NetBridge   string
	Kind        string
	SwitchID    string
	VmsCount    uint64
	ActivePorts uint64
	State       string
	Errors      []string
}

type NetworkCreateRequest struct {
	Name     string
	VLAN     uint32
	SwitchID string
}

type Host struct {
	ID       string
	Name     string
	Hostname string
	UUID     string
	Vendor   string
	Model    string
	Version  string
}

type VM struct {
	PowerState       string          `json:"-"`
	MonitoringState  uint32          `json:"-"`
	MonitoringStatus uint32          `json:"-"`
	DeploymentName   string          `json:"deployment_name,omitempty"`
	Name             string          `json:"name,omitempty"`
	UUID             string          `json:"uuid,omitempty"`
	Compatibility    string          `json:"compatibility,omitempty"`
	GuestOSFamily    string          `json:"guest_os_family,omitempty"`
	GuestOSVersion   string          `json:"guest_os_version,omitempty"`
	MachineType      string          `json:"machine_type,omitempty"`
	Storage          VMStorage       `json:"storage"`
	CPU              VMCPU           `json:"cpu"`
	Memory           VMMemory        `json:"memory"`
	VideoCard        VMVideo         `json:"video_card"`
	USBControllers   []USBController `json:"usb_controllers,omitempty"`
	InputDevices     []InputDevice   `json:"input_devices,omitempty"`
	DiskDevices      []DiskDevice    `json:"disk_devices,omitempty"`
	NetworkDevices   []NetworkDevice `json:"network_devices,omitempty"`
	Options          VMOptions       `json:"options"`
	IsTemplate       bool            `json:"is_template,omitempty"`
}

type VMStorage struct {
	ID     string `json:"id,omitempty"`
	Folder string `json:"folder,omitempty"`
}

type VMCPU struct {
	VCPUs          int32  `json:"vcpus,omitempty"`
	MaxVCPUs       int32  `json:"max_vcpus,omitempty"`
	CorePerSocket  int32  `json:"core_per_socket,omitempty"`
	Model          string `json:"model,omitempty"`
	ReservationMHz int32  `json:"reservation_mhz,omitempty"`
	LimitMHz       int32  `json:"limit_mhz,omitempty"`
	Shares         int32  `json:"shares,omitempty"`
	Hotplug        bool   `json:"hotplug,omitempty"`
}

type VMMemory struct {
	SizeMB        int32 `json:"size_mb,omitempty"`
	Hotplug       bool  `json:"hotplug,omitempty"`
	ReservationMB int32 `json:"reservation_mb,omitempty"`
	LimitMB       int32 `json:"limit_mb,omitempty"`
}

type VMVideo struct {
	Adapter  string `json:"adapter,omitempty"`
	Displays int32  `json:"displays,omitempty"`
	MemoryMB int32  `json:"memory_mb,omitempty"`
}

type InputDevice struct {
	Type string `json:"type,omitempty"`
	Bus  string `json:"bus,omitempty"`
}

type USBController struct {
	Type string `json:"type,omitempty"`
}

type DiskDevice struct {
	Size          uint64 `json:"size,omitempty"`
	Source        string `json:"source,omitempty"`
	StorageID     string `json:"storage_id,omitempty"`
	DeviceType    string `json:"device_type,omitempty"`
	Bus           string `json:"bus,omitempty"`
	Target        string `json:"target,omitempty"`
	BootOrder     int32  `json:"boot_order,omitempty"`
	ProvisionType string `json:"provision_type,omitempty"`
	DiskMode      string `json:"disk_mode,omitempty"`
	ReadOnly      bool   `json:"read_only,omitempty"`
	Create        bool   `json:"create,omitempty"`
	Remove        bool   `json:"remove,omitempty"`
	Attach        bool   `json:"attach,omitempty"`
	Detach        bool   `json:"detach,omitempty"`
	Resize        bool   `json:"resize,omitempty"`
}

type NetworkDevice struct {
	Network   string `json:"network,omitempty"`
	NetBridge string `json:"net_bridge,omitempty"`
	MAC       string `json:"mac,omitempty"`
	Target    string `json:"target,omitempty"`
	Model     string `json:"model,omitempty"`
	BootOrder int32  `json:"boot_order,omitempty"`
	VLAN      int32  `json:"vlan,omitempty"`
	Attach    bool   `json:"attach,omitempty"`
	Detach    bool   `json:"detach,omitempty"`
}

type VMOptions struct {
	RemoteConsole RemoteConsole `json:"remote_console"`
	GuestTools    GuestTools    `json:"guest_tools"`
	BootOptions   BootOptions   `json:"boot_options"`
}

type RemoteConsole struct {
	Type          string `json:"type,omitempty"`
	Port          int32  `json:"port,omitempty"`
	Keymap        string `json:"keymap,omitempty"`
	Password      string `json:"password,omitempty"`
	GuestOSLock   bool   `json:"guest_os_lock,omitempty"`
	LimitSessions int32  `json:"limit_sessions,omitempty"`
	Spice         Spice  `json:"spice"`
}

type Spice struct {
	ImgCompression      string `json:"img_compression,omitempty"`
	JpegCompression     string `json:"jpeg_compression,omitempty"`
	ZlibGlzCompression  string `json:"zlib_glz_compression,omitempty"`
	StreamingMode       string `json:"streaming_mode,omitempty"`
	PlaybackCompression bool   `json:"playback_compression,omitempty"`
	FileTransfer        bool   `json:"file_transfer,omitempty"`
	Clipboard           bool   `json:"clipboard,omitempty"`
}

type GuestTools struct {
	Enabled          bool `json:"enabled,omitempty"`
	SynchronizedTime bool `json:"synchronized_time,omitempty"`
	Balloon          bool `json:"balloon,omitempty"`
}

type BootOptions struct {
	Firmware    string `json:"firmware,omitempty"`
	BootDelayMS int32  `json:"boot_delay_ms,omitempty"`
	BootMenu    bool   `json:"boot_menu,omitempty"`
}

type ValidateResult struct {
	Errors []ValidationError
	VM     *VM
}

type ValidationError struct {
	Field   string
	Message string
}

type CreateVMRequest struct {
	Start bool
	VM    *VM
}

type Task struct {
	ID          string
	Method      string
	Status      uint32
	Error       string
	Target      string
	CreatedID   string
	CreatedName string
	CompletedAt *time.Time
}

type DeployTemplateRequest struct {
	TemplateID string
	Name       string
	StorageID  string
	Start      bool
	Linked     bool
	CloneCount int64
	PVM        *VM
}

type Snapshot struct {
	ID            int64
	Name          string
	Description   string
	Timestamp     int64
	Size          int64
	QuiesceFS     bool
	VMDescription string
	MemoryEnabled bool
	MemorySource  string
	ParentID      int32
	Disks         []SnapshotDisk
}

type SnapshotDisk struct {
	Source string
	Target string
}
