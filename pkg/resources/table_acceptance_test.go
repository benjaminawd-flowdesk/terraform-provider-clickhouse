package resources_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testResourceTableDatabaseName = "test_database"
const testResourceTableTableName = "replicated_table_test"

func TestAccResourceTable(t *testing.T) {

	resource.UnitTest(t, resource.TestCase{
		PreCheck: func() { testutils.TestAccPreCheck(t) },
		//ProviderFactories: ProviderFactories,
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: tableConfigWithName(testResourceTableDatabaseName, testResourceTableTableName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db_resource", "name", regexp.MustCompile("^"+testResourceTableDatabaseName)),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db_resource", "comment", regexp.MustCompile("^this is a comment")),

					resource.TestCheckResourceAttr("clickhouse_table.table", "name", testResourceTableTableName),
					resource.TestCheckResourceAttr("clickhouse_table.table", "database", testResourceTableDatabaseName),
					resource.TestCheckNoResourceAttr("clickhouse_table.table", "cluster"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "comment", "This is just a new table"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine", "ReplacingMergeTree"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine_params.#", "1"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine_params.0", "eventTime"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "order_by.#", "2"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "order_by.0", "key"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "order_by.1", "toStartOfHour(eventTime)"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "primary_key.#", "1"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "primary_key.0", "key"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.#", "3"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.0.name", "key"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.0.type", "Int64"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.1.name", "someCol"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.1.type", "String"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.name", "eventTime"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.type", "DateTime"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.default_kind", "DEFAULT"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.default_expression", "now()"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.compression_codec", "CODEC(DoubleDelta, ZSTD(1))"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "ttl.%", "2"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "ttl.toDateTime(eventTime)", "DELETE"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "ttl.toDateTime(eventTime) + INTERVAL 4 HOUR", "DELETE where key > 0"),
				),
			},
		},
	})
}

func tableConfigWithName(database string, tableName string) string {
	s := `
	resource "clickhouse_db" "new_db_resource" {
		name = "%_database_%"
		comment = "this is a comment"
	}

	resource "clickhouse_table" "table" {
		database = clickhouse_db.new_db_resource.name
		name = "%_tableName_%"
		engine = "ReplacingMergeTree"
		engine_params = ["eventTime"]
		order_by = ["key", "toStartOfHour(eventTime)"]
		primary_key = ["key"]
		column  {
			name= "key"
			type= "Int64"
		}
		column {
			name= "someCol"
			type= "String"
		}
		column {
			name= "eventTime"
			type= "DateTime"
			default_kind= "DEFAULT"
			default_expression= "now()"
			compression_codec= "CODEC(DoubleDelta, ZSTD(1))"
		}
		partition_by {
			by = "eventTime"
			partition_function = "toYYYYMM"
		}
		ttl = {
			"toDateTime(eventTime)" = "DELETE"
			"toDateTime(eventTime) + INTERVAL 4 HOUR" = "DELETE where key > 0"
		}
		comment = "This is just a new table"
}`

	s = strings.ReplaceAll(s, "%_database_%", database)
	s = strings.ReplaceAll(s, "%_tableName_%", tableName)
	return s
}

func TestGetCreateStatementForTable(t *testing.T) {
	testCases := []testutils.TestCase{
		{
			EnvVars: map[string]string{
				"TF_VAR_CREATE_OR_REPLACE": "true",
			},
			ExpectedSQL: "CREATE OR REPLACE TABLE",
		},
		{
			EnvVars: map[string]string{
				"TF_VAR_CREATE_IF_NOT_EXISTS": "true",
			},
			ExpectedSQL: "CREATE TABLE IF NOT EXISTS",
		},
		{
			EnvVars:     map[string]string{}, // Default case
			ExpectedSQL: "CREATE TABLE",
		},
		{
			EnvVars: map[string]string{
				"TF_VAR_CREATE_IF_NOT_EXISTS": "true",
				"TF_VAR_CREATE_OR_REPLACE":    "true",
			},
			ExpectedSQL: "CREATE OR REPLACE TABLE", // Priority for CREATE OR REPLACE table
		},
	}

	testutils.RunGetCreateStatementTest(t, "TABLE", testCases)
}
