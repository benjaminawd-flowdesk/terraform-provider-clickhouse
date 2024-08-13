package resources_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/common"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/models"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type TestRoleStepData struct {
	roleName   string
	database   string
	privileges []string
}

const roleResourceName = "test_role"
const roleResource = "clickhouse_role." + roleResourceName
const roleName1 = "test_role_1"
const roleName2 = "test_role_2"
const databaseName1 = "role_role_db_1"
const databaseName2 = "role_role_db_2"

var test1StepsData = []TestRoleStepData{
	{
		// Create role
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"SELECT",
			"INSERT",
		},
	},
	{
		// Remove role privileges
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"SELECT",
		},
	},
	{
		// Remove and add role privileges
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"INSERT",
		},
	},
	{
		// Update db name
		roleName: roleName1,
		database: databaseName2,
		privileges: []string{
			"INSERT",
		},
	},
	{
		// Update db name and privileges at the same time
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"INSERT",
			"ALTER",
		},
	},
	{
		// Check all allowed privileges
		roleName:   roleName1,
		database:   databaseName1,
		privileges: resources.AllowedDbLevelPrivileges,
	},
	{
		// Change role name
		roleName:   roleName2,
		database:   databaseName1,
		privileges: resources.AllowedDbLevelPrivileges,
	},
	{
		// Change role name and db
		roleName:   roleName1,
		database:   databaseName2,
		privileges: resources.AllowedDbLevelPrivileges,
	},
	{
		// Change role name, db and privileges
		roleName: roleName2,
		database: databaseName1,
		privileges: []string{
			"INSERT",
		},
	},
}

var test2StepsData = []TestRoleStepData{
	{
		// Create role
		roleName: roleName1,
		database: "system",
		privileges: []string{
			"SELECT",
			"INSERT",
		},
	},
	{
		// Remove role privileges
		roleName: roleName1,
		database: "system",
		privileges: []string{
			"SELECT",
		},
	},
	{
		// Remove and add role privileges
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"INSERT",
		},
	},
	{
		// Check all allowed privileges
		roleName:   roleName1,
		database:   "system",
		privileges: resources.AllowedDbLevelPrivileges,
	},
	{
		// Change role name
		roleName:   roleName2,
		database:   "system",
		privileges: resources.AllowedDbLevelPrivileges,
	},
}

func generateRoleTestSteps(testStepsData []TestRoleStepData) []resource.TestStep {
	var testSteps []resource.TestStep
	for _, testStepData := range testStepsData {
		var databaseRegex *regexp.Regexp
		if testStepData.database == "*" {
			databaseRegex = regexp.MustCompile(`\*`)
		} else {
			databaseRegex = regexp.MustCompile(testStepData.database)
		}
		testSteps = append(testSteps, resource.TestStep{
			Config: testAccRoleResource(
				testStepData.roleName,
				testStepData.database,
				common.Quote(testStepData.privileges),
			),
			Check: resource.ComposeTestCheckFunc(
				resource.TestMatchResourceAttr(
					roleResource,
					"name",
					regexp.MustCompile(testStepData.roleName),
				),
				resource.TestMatchResourceAttr(
					roleResource,
					"database",
					databaseRegex,
				),
				testutils.CheckStateSetAttr("privileges", roleResource, testStepData.privileges),
				testAccCheckRoleResourceExists(testStepData.roleName, testStepData.database, testStepData.privileges),
			),
		})
	}
	return testSteps
}

func TestAccResourceRole(t *testing.T) {
	// Feature tests, user database
	resource.Test(t, resource.TestCase{
		//ProviderFactories: testutils.GetProviderFactories(),
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps:        generateRoleTestSteps(test1StepsData),
	})
	// Feature tests, system database
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps:        generateRoleTestSteps(test2StepsData),
	})
	// Feature tests, global privileges
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps: generateRoleTestSteps([]TestRoleStepData{
			{
				// Create role
				roleName: roleName1,
				database: "*",
				privileges: []string{
					"REMOTE",
				},
			}}),
	})
	// Validate privileges on create
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"NOT_ALLOWED_PRIVILEGE"}),
				),
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE isn't in the allowed privileges list"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"REMOTE"}),
				),
				ExpectError: regexp.MustCompile(`Global privilege REMOTE is only allowed for database '*'`),
			},
		},
	})
	// Validate privileges on update
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1}),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"SELECT"}),
				),
			},
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"NOT_ALLOWED_PRIVILEGE"}),
				),
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE isn't in the allowed privileges list"),
			},
		},
	})
}

func testAccRoleResource(roleName string, database string, privileges []string) string {
	if database == "system" {
		return fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
		database = "system"
		privileges = [%s]
	}`, roleName, strings.Join(privileges, ","))
	}

	if database == "*" {
		return fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
		database = "*"
		privileges = [%s]
	}`, roleName, strings.Join(privileges, ","))
	}

	databaseComment := "db comment"
	databaseResource := fmt.Sprintf(`
	resource "clickhouse_db" "%[1]s" {
		name = "%[1]s"
		comment = "%[3]s"
	}

	resource "clickhouse_db" "%[2]s" {
		name = "%[2]s"
		comment = "%[3]s"
	}
`, databaseName1, databaseName2, databaseComment)

	roleResource := fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
		database = clickhouse_db.%s.name
		privileges = [%s]
	}
`, roleName, database, strings.Join(privileges, ","))

	return fmt.Sprintf("%s\n%s", databaseResource, roleResource)
}

func testAccCheckRoleResourceExists(roleName string, database string, privileges []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*sdk.Client)

		dbRole, err := client.GetRole(context.Background(), roleName)

		if err != nil {
			return fmt.Errorf("get role: %v", err)
		}
		if dbRole == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		if len(privileges) != len(dbRole.Privileges) {
			return fmt.Errorf("role privileges length mismatching between db and state")
		}

		for _, privilege := range privileges {
			var matchedDbRolePrivilege *models.CHGrant
			for _, dbRolePrivilege := range dbRole.Privileges {
				if privilege == dbRolePrivilege.AccessType {
					matchedDbRolePrivilege = &dbRolePrivilege
					break
				}
			}
			if matchedDbRolePrivilege == nil {
				return fmt.Errorf("role privilege %s not found in db", privilege)
			}
			if matchedDbRolePrivilege.Database != database {
				return fmt.Errorf("role privilege %s database mismatching between db and state", privilege)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceDestroy(roleNames []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, roleName := range roleNames {
			client := testutils.TestAccProvider.Meta().(*sdk.Client)

			dbRole, err := client.GetRole(context.Background(), roleName)

			if err != nil {
				return fmt.Errorf("get role: %v", err)
			}

			if dbRole != nil {
				return fmt.Errorf("role %s hasn't been deleted", roleName)
			}
		}
		return nil
	}
}
