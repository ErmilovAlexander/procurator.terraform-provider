package client

import "context"

type UmbraClient interface {
	ListNetworks(context.Context, string) ([]Network, error)
	GetNetworkByID(context.Context, string) (*Network, error)
	GetNetworkByName(context.Context, string) (*Network, error)
	CreateNetwork(context.Context, *NetworkCreateRequest) (string, error)
	UpdateNetwork(context.Context, *Network) error
	DeleteNetwork(context.Context, string) error
	ListSwitches(context.Context) ([]Switch, error)
	GetSwitch(context.Context, string) (*Switch, error)
	CreateSwitch(context.Context, *SwitchCreateRequest) (string, error)
	UpdateSwitch(context.Context, *Switch) error
	DeleteSwitch(context.Context, string) error
	ListNICs(context.Context, string) ([]NIC, error)
}

type StorageClient interface {
	ListStorageAdapters(context.Context) ([]StorageAdapter, error)
	ListStorageDevices(context.Context) ([]StorageDevice, error)
}

type Client interface {
	AccessToken() string
	ListTemplates(context.Context) ([]VM, error)
	GetTemplate(context.Context, string) (*VM, error)
	CreateTemplateFromVM(context.Context, string, string, string) (string, error)
	DeleteTemplate(context.Context, string) (string, error)
	DeployTemplate(context.Context, *DeployTemplateRequest) (string, error)
	ConvertVMToTemplate(context.Context, string) (string, error)
	ConvertTemplateToVM(context.Context, string) (string, error)
	GetHost(context.Context) (*Host, error)
	ListDatastores(context.Context) ([]Datastore, error)
	GetDatastore(context.Context, string) (*Datastore, error)
	CreateDatastore(context.Context, *Datastore) (string, error)
	DeleteDatastore(context.Context, string) (string, error)
	BrowseDatastoreItem(context.Context, string) (*DatastoreItem, error)
	CreateDatastoreFolder(context.Context, string) (string, error)
	DeleteDatastoreItem(context.Context, []string) (string, error)
	ListVMs(context.Context) ([]VM, error)
	GetVM(context.Context, string) (*VM, error)
	GetVMWithParams(context.Context, string, bool) (*VM, error)
	ValidateVM(context.Context, *VM) (*ValidateResult, error)
	CreateVM(context.Context, *CreateVMRequest) (string, error)
	UpdateVM(context.Context, *VM) (string, error)
	DeleteVM(context.Context, string) (string, error)
	SetVMPowerState(context.Context, string, string, bool) (string, error)
	WaitTask(context.Context, string) (*Task, error)
	MigrateVMDatastore(context.Context, string, map[string]VMDatastoreMigrationItem) (string, error)
	ListVMSnapshots(context.Context, string) ([]Snapshot, int64, error)
	TakeVMSnapshot(context.Context, string, string, string, bool, bool) (string, error)
	DeleteVMSnapshot(context.Context, string, int64) (string, error)
}
