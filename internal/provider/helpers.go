package provider

import (
	"context"

	"procurator.terraform-provider/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringValue(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func int64Value(v int64) types.Int64 { return types.Int64Value(v) }
func boolValue(v bool) types.Bool    { return types.BoolValue(v) }

func boolFrom(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func stringList(ctx context.Context, vals []string) types.List {
	res, diags := types.ListValueFrom(ctx, types.StringType, vals)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return res
}

func selectVM(items []client.VM, id, deploymentName, name, uuid string) *client.VM {
	for i := range items {
		it := items[i]
		switch {
		case id != "" && it.DeploymentName == id:
			return &it
		case deploymentName != "" && it.DeploymentName == deploymentName:
			return &it
		case uuid != "" && it.UUID == uuid:
			return &it
		case name != "" && it.Name == name:
			return &it
		}
	}
	return nil
}

func selectDatastore(items []client.Datastore, id, name string) *client.Datastore {
	for i := range items {
		it := items[i]
		if id != "" && it.ID == id {
			return &it
		}
		if name != "" && it.Name == name {
			return &it
		}
	}
	return nil
}
