terraform {
  required_providers {
    clickhouse = {
      version = "2.0.0"
      source  = "hashicorp.com/flowdeskmarkets/clickhouse"
    }
  }
}


provider "clickhouse" {
  port     = 8123
  host     = "127.0.0.1"
  username = "default"
  password = ""
}
