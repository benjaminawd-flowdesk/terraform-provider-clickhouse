---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "clickhouse_dbs Data Source - terraform-provider-clickhouse"
subcategory: ""
description: |-
  Datasource to retrieve all databases set in clickhouse instance
---

# clickhouse_dbs (Data Source)

Datasource to retrieve all databases set in clickhouse instance



<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `dbs` (List of Object) (see [below for nested schema](#nestedatt--dbs))
- `id` (String) The ID of this resource.

<a id="nestedatt--dbs"></a>
### Nested Schema for `dbs`

Read-Only:

- `comment` (String)
- `data_path` (String)
- `engine` (String)
- `metadata_path` (String)
- `name` (String)
- `uuid` (String)
