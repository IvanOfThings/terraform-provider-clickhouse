//go:build testing

package testutils

import (
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TestAccProviders map[string]*schema.Provider
var TestAccProvider *schema.Provider

func init() {
	TestAccProvider = provider.New("dev")()
	TestAccProviders = map[string]*schema.Provider{
		"clickhouse": TestAccProvider,
	}
}

func Provider() map[string]*schema.Provider {
	return TestAccProviders
}

func TestAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}
