package testutils

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"strconv"
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

//func ClickhouseProviderFactory() (*schema.Provider, error) {
//	TestAccProvider = provider.New("dev")()
//	return TestAccProvider, nil
//}
//
//func GetProviderFactories() map[string]func() (*schema.Provider, error) {
//	return map[string]func() (*schema.Provider, error){"clickhouse": ClickhouseProviderFactory}
//}

func getSetFromSetStateAttribute(attributes map[string]string, key string) (*schema.Set, error) {
	itemsLength, err := strconv.Atoi(attributes[fmt.Sprintf("%s.#", key)])
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for i := 0; i < itemsLength; i++ {
		items = append(items, attributes[fmt.Sprintf("%s.%d", key, i)])
	}

	if len(items) != itemsLength {
		return nil, fmt.Errorf("%s.# mismatch items retrieved from state", key)
	}

	return schema.NewSet(schema.HashString, items), nil
}

// Elements of lists and sets are stored in terraform state as independent variables with the sufix "_{index}".
// To check the value, we need to get the length of the list and iterate over it and get the list elements one by one
// and compare them with the values from the current test plan.
func CheckStateSetAttr(attrKey string, resource string, expectedItems []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		attributes := state.RootModule().Resources[resource].Primary.Attributes
		set, err := getSetFromSetStateAttribute(attributes, attrKey)
		if err != nil {
			return fmt.Errorf("get %s set: %v", attrKey, err)
		}
		if len(expectedItems) != set.Len() {
			return fmt.Errorf("expectedItems length mismatching between plan and state")
		}

		for _, expectedItem := range expectedItems {
			if set.Contains(expectedItem) == false {
				return fmt.Errorf("expectedItem %s not found in state", expectedItem)
			}
		}

		return nil
	}
}
