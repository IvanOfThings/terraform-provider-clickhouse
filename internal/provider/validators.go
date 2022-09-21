package clickhouse_provider

import (
	"errors"
	"fmt"

	v "github.com/go-playground/validator/v10"
	hashicorpcty "github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func validateParams(mappedColumns []CHColumn, params []string, params_name string) error {
	for _, param := range params {
		found := false
		for _, column := range mappedColumns {
			if column.name == param {
				found = true
				break
			}
		}
		if found == false {
			err := errors.New(fmt.Sprintf("Value %v not found in columns a value in paramter %v", param, params_name))
			return err
		}
	}
	return nil
}

func validatePartitionBy(inValue any, p hashicorpcty.Path) diag.Diagnostics {
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

func validateType(inValue any, p hashicorpcty.Path) diag.Diagnostics {
	validate := v.New()
	value := inValue.(string)
	uintTypes := "UInt8 UInt16 UInt32 UInt64 UInt128 UInt256 Int8 Int16 Int32 Int64 Int128 Int256 Float32 Float64"
	otherTypes := "Bool String UUID Date Date32 Datetime Datetime64 LowCardinality JSON"
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

func validateOnClusterEngine(inValue any, p hashicorpcty.Path) diag.Diagnostics {
	validate := v.New()
	value := inValue.(string)
	replicatedTypes := "ReplicatedMergeTree"
	distributedTypes := "Distributed"
	validation := fmt.Sprintf("oneof=%v %v %v", replicatedTypes, distributedTypes)
	var diags diag.Diagnostics
	if validate.Var(value, validation) != nil {
		diag := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail:   fmt.Sprintf("%q is not %q %q %q", value, replicatedTypes, distributedTypes),
		}
		diags = append(diags, diag)
	}
	return diags
}
