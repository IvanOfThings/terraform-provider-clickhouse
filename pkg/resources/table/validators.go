package resourcetable

import (
	"fmt"

	v "github.com/go-playground/validator/v10"
	hashicorpcty "github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func ValidatePartitionBy(inValue any, p hashicorpcty.Path) diag.Diagnostics {
	validate := v.New()
	value := inValue.(string)
	toAllowedPartitioningFunctions := "toYYYYMM toYYYYMMDD toYYYYMMDDhhmmss"
	validation := fmt.Sprintf("oneof=%v", toAllowedPartitioningFunctions)
	var diags diag.Diagnostics
	if validate.Var(value, validation) != nil {
		diag := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail:   fmt.Sprintf("%q is not %q", value, toAllowedPartitioningFunctions),
		}
		diags = append(diags, diag)
	}
	return diags
}

func ValidateType(inValue any, p hashicorpcty.Path) diag.Diagnostics {
	validate := v.New()
	value := inValue.(string)
	uintTypes := "UInt8 UInt16 UInt32 UInt64 UInt128 UInt256 Int8 Int16 Int32 Int64 Int128 Int256 Float32 Float64"
	otherTypes := "Bool String UUID Date Date32 DateTime DateTime64 LowCardinality JSON"
	validation := fmt.Sprintf("oneof=%v %v", uintTypes, otherTypes)
	var diags diag.Diagnostics
	if validate.Var(value, validation) != nil {
		diag := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail:   fmt.Sprintf("%q is not %q %q", value, uintTypes, otherTypes),
		}
		diags = append(diags, diag)
	}
	return diags
}

func ValidateOnClusterEngine(inValue any, p hashicorpcty.Path) diag.Diagnostics {
	validate := v.New()
	value := inValue.(string)
	mergeTreeTypes := "ReplacingMergeTree"
	replicatedTypes := "ReplicatedMergeTree"
	distributedTypes := "Distributed"
	validation := fmt.Sprintf("oneof=%v %v %v", replicatedTypes, distributedTypes, mergeTreeTypes)
	var diags diag.Diagnostics
	if validate.Var(value, validation) != nil {
		diag := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail:   fmt.Sprintf("%q is not %q %q %q", value, replicatedTypes, distributedTypes, mergeTreeTypes),
		}
		diags = append(diags, diag)
	}
	return diags
}
