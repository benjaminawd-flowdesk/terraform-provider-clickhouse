package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TODO: Testing trying to delete a db that have other resources on it (like database.)

func TestAccResourceDb(t *testing.T) {

	resource.UnitTest(t, resource.TestCase{
		PreCheck: func() { testutils.TestAccPreCheck(t) },
		//ProviderFactories: ProviderFactories,
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: dbConfig(testResourceDBDatabaseName, testResourceDBDatabaseComment),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "name", regexp.MustCompile("^"+testResourceDBDatabaseName)),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "comment", regexp.MustCompile("^"+testResourceDBDatabaseComment)),
				),
			},
			// RECREATE WITH A DIFFERENT NAME
			{
				Config: dbConfig(testResourceDBDatabaseName2, testResourceDBDatabaseComment),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "name", regexp.MustCompile("^"+testResourceDBDatabaseName2)),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "comment", regexp.MustCompile("^"+testResourceDBDatabaseComment)),
				),
			},
			// RECREATE WITH A DIFFERENT COMMENT
			{
				Config: dbConfig(testResourceDBDatabaseName2, testResourceDBDatabaseComment2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "name", regexp.MustCompile("^"+testResourceDBDatabaseName2)),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "comment", regexp.MustCompile("^"+testResourceDBDatabaseComment2)),
				),
			},
			{
				ResourceName:            "clickhouse_db.new_db",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"comment"},
			},
		},
	})
}

func TestGetCreateStatementForDatabase(t *testing.T) {
	testCases := []testutils.TestCase{
		{
			EnvVars: map[string]string{
				"TF_VAR_CREATE_IF_NOT_EXISTS": "true",
			},
			ExpectedSQL: "CREATE DATABASE IF NOT EXISTS",
		},
		{
			EnvVars:     map[string]string{}, // Default case
			ExpectedSQL: "CREATE DATABASE",
		},
		{
			EnvVars: map[string]string{
				"TF_VAR_CREATE_IF_NOT_EXISTS": "true",
				"TF_VAR_CREATE_OR_REPLACE":    "true",
			},
			ExpectedSQL: "CREATE DATABASE IF NOT EXISTS", // Replace env var has no effect
		},
	}

	testutils.RunGetCreateStatementTest(t, "database", testCases)
}

const testResourceDBDatabaseName = "testing_db"
const testResourceDBDatabaseName2 = "testing_db_2"
const testResourceDBDatabaseComment = "This is a testing database"
const testResourceDBDatabaseComment2 = "This is a testing database 2"

func dbConfig(databaseName string, comment string) string {
	s := `
	resource "clickhouse_db" "new_db" {
		name = "%v"
		comment = "%v"
	}
`
	return fmt.Sprintf(s, databaseName, comment)
}
