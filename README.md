# Terraform Provider: Clickhouse (Terraform Plugin SDK)

_This template repository is built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk). The template repository built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) can be found at [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework). See [Which SDK Should I Use?](https://www.terraform.io/docs/plugin/which-sdk.html) in the Terraform documentation for additional information._

----

This is a terraform provider plugin for managing Clickhouse databases and tables in a simple way.

_Note_: This provider it's in a very early state so only few table engines are allowed for replicated tables so far.


## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:
```sh
$ go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```bash
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider


Definining provider. The port should be the Clickhouse native protocol port (9000 by default, and 9440 for Clickhouse Cloud)

```hcl
provider "clickhouse" {
  port           = 9000           # Clickhouse native protocol port
  host           = "127.0.0.1"
  username       = "default"
  password       = ""
}
```

In order to define the host, username and password in a safety way it is possible to define them using env vars:

```config
TF_VAR_CLICKHOUSE_USERNAME=default
TF_VAR_CLICKHOUSE_PASSWORD=""
TF_VAR_CLICKHOUSE_HOST="127.0.0.1"
TF_VAR_CLICKHOUSE_PORT=9000
```

```hcl
resource "clickhouse_db" "test_db_clusterd" {
  name = "database_test_clustered"
  comment = "This is a test database"
}
```

### Creating or replacing tables

It is possible to modify the CREATE TABLE/CREATE DATABASE statement using the following variables:

```config
TF_VAR_CREATE_OR_REPLACE=true
```
or

```config
TF_VAR_CREATE_IF_NOT_EXISTS=true
```

### Clustered server

Configuring provider

```hcl
provider "clickhouse" {
  port           = 9000
  host           = "127.0.0.1"
  username       = "default"
  password       = ""
  default_cluster ="cluster"
}
```

Creating a Database

```hcl
resource "clickhouse_db" "test_db_clusterd" {
  name = "database_test_clustered"
  comment = "This is a test database"
  cluster = "cluster"
}
```

### Clustered server using Altinity Clickhouse Operator

It is possible to use macros defined for cluster, databases, installation names in Altinity operator when creating resources.

```hcl
provider "clickhouse" {
  port           = 9000
  host           = "127.0.0.1"
  username       = "default"
  password       = ""
  default_cluster ="'{cluster}'"
}
```

```hcl
resource "clickhouse_db" "test_db_cluster" {
  name = "database_test_clustered"
  comment = "This is a test database"
  cluster = "'{cluster}'"
}
```

Creating tables

```hcl
resource "clickhouse_table" "replicated_table" {
  database      = clickhouse_db.test_db_clustered.name
  name    = "replicated_table"
  cluster       = clickhouse_db.test_db_clustered.cluster
  engine        = "ReplicatedMergeTree"
  engine_params = ["'/clickhouse/{installation}/clickhouse_db.test_db_clustered.cluster/tables/{shard}/{database}/{table}'", "'{replica}'"]
  order_by      = ["event_date", "event_type"]
  columns {
    name = "event_date"
    type = "Date"
  }
  columns {
    name = "event_type"
    type = "Int32"
  }
  columns {
    name = "article_id"
    type = "Int32"
  }
  columns {
    name = "title"
    type = "String"
  }
  partition_by {
    by = "event_type"
  }
  partition_by {
    by                 = "event_date"
    partition_function = "toYYYYMM"
  }
}


resource "clickhouse_table" "distributed_table" {
  database      = clickhouse_db.test_db_clustered.name
  name    = "distributed_table"
  cluster       = clickhouse_db.test_db_clustered.cluster
  engine        = "Distributed"
  engine_params = [clickhouse_db.test_db_clustered.cluster, clickhouse_db.test_db_clustered.name, clickhouse_table.replicated_table.name, "rand()"]
}
```

Creating roles

```hcl
resource "clickhouse_role" "my_database_rw" {
  name       = "my_database_rw"
  database   = clickhouse_db.test_db_cluster.name
  privileges = ["SELECT", "INSERT"]
}
```

Creating users

```hcl
resource "clickhouse_user" "my_database_rw_user" {
  name     = "my_database_rw_user"
  password = "awesome_user_password"
  roles    = [clickhouse_role.my_database_rw.name]
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

After making changes to provider, run `go build -o terraform-provider-clickhouse` to create a local binary.

Create a .terraformrc config file in `$HOME/.terraformrc`:

```terraform
provider_installation {

  dev_overrides {
      "hashicorp.com/flowdeskmarkets/clickhouse" = "/Users/<path-to-binary>/terraform-provider-clickhouse"
  }
  direct {}
}
```

> [!WARNING]
> Disable this before running commands on production to avoid using an untested version of the provider.

To generate or update documentation, run `go generate`.

To run tests locally, you can run a local Clickhouse server:

```sh
curl https://clickhouse.com/ | sh
CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1 ./clickhouse server
```

then run the tests:

```sh
$ go build -o terraform-clickhouse-provider && make testacc
```

To release the provider on the Hashicorp registry, manually trigger the `release` workflow via Github.

Ensure that the current tag was made against the latest commit, otherwise you may face an error like:

```error
  ⨯ release failed after 0s                  error=git tag v1.2.0 was not made against commit ad38fef5
```
