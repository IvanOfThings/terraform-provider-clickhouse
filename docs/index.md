---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "clickhouse Provider"
subcategory: ""
description: |-
  
---

# clickhouse Provider



## Example Usage

```terraform
terraform {
  required_providers {
    clickhouse = {
      version = "3.0.1"
      source  = "IvanOfThings/clickhouse"
    }
  }
}


provider "clickhouse" {
  port     = 8123
  host     = "127.0.0.1"
  username = "default"
  password = ""
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `default_cluster` (String) Default cluster, if provided will be used when no cluster is provided
- `host` (String, Sensitive) Clickhouse server url
- `password` (String, Sensitive) Clickhouse user password with admin privileges
- `port` (Number) Clickhouse server native protocol port (TCP)
- `secure` (Boolean) Clickhouse secure connection
- `username` (String) Clickhouse username with admin privileges
