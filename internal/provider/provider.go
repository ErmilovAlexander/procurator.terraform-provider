package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ProcuratorProvider{}

type ProcuratorProvider struct {
	version string
}

type providerModel struct {
	Endpoint        types.String `tfsdk:"endpoint"`
	UmbraEndpoint   types.String `tfsdk:"umbra_endpoint"`
	StorageEndpoint types.String `tfsdk:"storage_endpoint"`
	Token           types.String `tfsdk:"token"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	Insecure        types.Bool   `tfsdk:"insecure"`
	CAFile          types.String `tfsdk:"ca_file"`
	Authority       types.String `tfsdk:"authority"`
}

type providerData struct {
	client      client.Client
	umbra       client.UmbraClient
	umbraCfg    client.Config
	umbraOnce   sync.Once
	umbraErr    error
	storage     client.StorageClient
	storageCfg  client.Config
	storageOnce sync.Once
	storageErr  error
}

func (d *providerData) getUmbra() (client.UmbraClient, error) {
	d.umbraOnce.Do(func() {
		d.umbra, d.umbraErr = client.NewUmbra(d.umbraCfg)
	})
	return d.umbra, d.umbraErr
}

func (d *providerData) getStorage() (client.StorageClient, error) {
	d.storageOnce.Do(func() {
		d.storage, d.storageErr = client.NewStorage(d.storageCfg)
	})
	return d.storage, d.storageErr
}

func (d *providerData) ListNetworks(ctx context.Context, filter string) ([]client.Network, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.ListNetworks(ctx, filter)
}

func (d *providerData) GetNetworkByID(ctx context.Context, id string) (*client.Network, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.GetNetworkByID(ctx, id)
}

func (d *providerData) GetNetworkByName(ctx context.Context, name string) (*client.Network, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.GetNetworkByName(ctx, name)
}

func (d *providerData) CreateNetwork(ctx context.Context, req *client.NetworkCreateRequest) (string, error) {
	u, err := d.getUmbra()
	if err != nil {
		return "", err
	}
	return u.CreateNetwork(ctx, req)
}

func (d *providerData) UpdateNetwork(ctx context.Context, net *client.Network) error {
	u, err := d.getUmbra()
	if err != nil {
		return err
	}
	return u.UpdateNetwork(ctx, net)
}

func (d *providerData) DeleteNetwork(ctx context.Context, id string) error {
	u, err := d.getUmbra()
	if err != nil {
		return err
	}
	return u.DeleteNetwork(ctx, id)
}

func (d *providerData) ListSwitches(ctx context.Context) ([]client.Switch, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.ListSwitches(ctx)
}

func (d *providerData) GetSwitch(ctx context.Context, id string) (*client.Switch, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.GetSwitch(ctx, id)
}

func (d *providerData) CreateSwitch(ctx context.Context, req *client.SwitchCreateRequest) (string, error) {
	u, err := d.getUmbra()
	if err != nil {
		return "", err
	}
	return u.CreateSwitch(ctx, req)
}

func (d *providerData) UpdateSwitch(ctx context.Context, sw *client.Switch) error {
	u, err := d.getUmbra()
	if err != nil {
		return err
	}
	return u.UpdateSwitch(ctx, sw)
}

func (d *providerData) DeleteSwitch(ctx context.Context, id string) error {
	u, err := d.getUmbra()
	if err != nil {
		return err
	}
	return u.DeleteSwitch(ctx, id)
}

func (d *providerData) ListNICs(ctx context.Context, filter string) ([]client.NIC, error) {
	u, err := d.getUmbra()
	if err != nil {
		return nil, err
	}
	return u.ListNICs(ctx, filter)
}

func (d *providerData) ListStorageAdapters(ctx context.Context) ([]client.StorageAdapter, error) {
	s, err := d.getStorage()
	if err != nil {
		return nil, err
	}
	return s.ListStorageAdapters(ctx)
}

func (d *providerData) ListStorageDevices(ctx context.Context) ([]client.StorageDevice, error) {
	s, err := d.getStorage()
	if err != nil {
		return nil, err
	}
	return s.ListStorageDevices(ctx)
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ProcuratorProvider{version: version}
	}
}

func (p *ProcuratorProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "procurator"
	resp.Version = p.version
}

func (p *ProcuratorProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint":         schema.StringAttribute{Optional: true, Description: "gRPC endpoint of procurator.core, for example host:3641"},
			"umbra_endpoint":   schema.StringAttribute{Optional: true, Description: "gRPC endpoint of procurator.umbra, defaults to provider endpoint host with port 50051"},
			"storage_endpoint": schema.StringAttribute{Optional: true, Description: "gRPC endpoint of procurator.storage, defaults to provider endpoint host with port 3642"},
			"token":            schema.StringAttribute{Optional: true, Sensitive: true, Description: "Bearer token for procurator.core"},
			"username":         schema.StringAttribute{Optional: true, Description: "Username for Auth.Login when token is not supplied"},
			"password":         schema.StringAttribute{Optional: true, Sensitive: true, Description: "Password for Auth.Login when token is not supplied"},
			"insecure":         schema.BoolAttribute{Optional: true, Description: "Use TLS but skip certificate verification. Do not use in production."},
			"ca_file":          schema.StringAttribute{Optional: true, Description: "Path to CA certificate in PEM format used to verify procurator.core TLS certificate."},
			"authority":        schema.StringAttribute{Optional: true, Description: "TLS server name / :authority override, for example 127.0.0.1."},
		},
	}
}

func (p *ProcuratorProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := client.Config{
		Endpoint:        os.Getenv("PROCURATOR_ENDPOINT"),
		UmbraEndpoint:   os.Getenv("PROCURATOR_UMBRA_ENDPOINT"),
		StorageEndpoint: os.Getenv("PROCURATOR_STORAGE_ENDPOINT"),
		Token:           os.Getenv("PROCURATOR_TOKEN"),
		Username:        os.Getenv("PROCURATOR_USERNAME"),
		Password:        os.Getenv("PROCURATOR_PASSWORD"),
		Insecure:        false,
		CAFile:          os.Getenv("PROCURATOR_CA_FILE"),
		Authority:       os.Getenv("PROCURATOR_AUTHORITY"),
	}

	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		cfg.Endpoint = config.Endpoint.ValueString()
	}
	if !config.UmbraEndpoint.IsNull() && !config.UmbraEndpoint.IsUnknown() {
		cfg.UmbraEndpoint = config.UmbraEndpoint.ValueString()
	}
	if !config.StorageEndpoint.IsNull() && !config.StorageEndpoint.IsUnknown() {
		cfg.StorageEndpoint = config.StorageEndpoint.ValueString()
	}
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		cfg.Token = config.Token.ValueString()
	}
	if !config.Username.IsNull() && !config.Username.IsUnknown() {
		cfg.Username = config.Username.ValueString()
	}
	if !config.Password.IsNull() && !config.Password.IsUnknown() {
		cfg.Password = config.Password.ValueString()
	}
	if !config.Insecure.IsNull() && !config.Insecure.IsUnknown() {
		cfg.Insecure = config.Insecure.ValueBool()
	}
	if !config.CAFile.IsNull() && !config.CAFile.IsUnknown() {
		cfg.CAFile = config.CAFile.ValueString()
	}
	if cfg.Authority == "" {
		cfg.Authority = "127.0.0.1"
	}
	if !config.Authority.IsNull() && !config.Authority.IsUnknown() {
		cfg.Authority = config.Authority.ValueString()
	}

	if !strings.Contains(cfg.Endpoint, ":") {
		cfg.Endpoint = cfg.Endpoint + ":3641"
	}

	if cfg.Endpoint == "" {
		resp.Diagnostics.AddError("Missing endpoint", "Set provider.endpoint or PROCURATOR_ENDPOINT")
		return
	}
	if cfg.Authority == "" {
		cfg.Authority = "127.0.0.1"
	}

	if cfg.UmbraEndpoint == "" {
		parts := strings.Split(cfg.Endpoint, ":")
		if len(parts) >= 2 {
			host := strings.Join(parts[:len(parts)-1], ":")
			cfg.UmbraEndpoint = host + ":50051"
		} else {
			cfg.UmbraEndpoint = cfg.Endpoint
		}
	}
	if cfg.StorageEndpoint == "" {
		parts := strings.Split(cfg.Endpoint, ":")
		if len(parts) >= 2 {
			host := strings.Join(parts[:len(parts)-1], ":")
			cfg.StorageEndpoint = host + ":3642"
		} else {
			cfg.StorageEndpoint = cfg.Endpoint
		}
	}
	if cfg.Token == "" && cfg.Username == "" {
		resp.Diagnostics.AddError("Missing credentials", "Set token or username/password for procurator provider")
		return
	}

	cli, err := client.New(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to initialize Procurator client", fmt.Sprintf("endpoint=%s: %v", cfg.Endpoint, err))
		return
	}
	if cfg.Token == "" {
		cfg.Token = cli.AccessToken()
	}

	data := &providerData{client: cli, umbraCfg: cfg, storageCfg: cfg}
	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *ProcuratorProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewVMResource, NewDatastoreResource, NewDatastoreLVMResource, NewDatastoreFolderResource, NewTemplateResource, NewVMConvertToTemplateResource, NewVMSnapshotResource, NewVMDiskAttachmentResource, NewVMNetworkAttachmentResource, NewVMDatastoreMigrationResource, NewNetworkResource, NewSwitchResource}
}

func (p *ProcuratorProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{NewHostDataSource, NewDatastoreDataSource, NewVMDataSource, NewTemplateDataSource, NewNetworkDataSource, NewNetworksDataSource, NewSwitchDataSource, NewSwitchesDataSource, NewNICsDataSource, NewStorageAdaptersDataSource, NewStorageDevicesDataSource, NewVMSnapshotsDataSource}
}
