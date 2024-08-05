package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/common"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/datasources"
	resourcedb "github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources/db"
	resourcerole "github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources/role"
	resourcetable "github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources/table"
	resourceuser "github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources/user"
	resourceview "github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/resources/view"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/joho/godotenv"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"default_cluster": {
					Description: "Default cluster, if provided will be used when no cluster is provided",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
				},
				"username": {
					Description: "Clickhouse username with admin privileges",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_USERNAME")
					},
				},
				"password": {
					Description: "Clickhouse user password with admin privileges",
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: func() (any, error) {
						if password, _ := getEnvVar("TF_CLICKHOUSE_PASSWORD"); password != nil {
							return password, nil
						}
						return "", nil
					},
				},
				"host": {
					Description: "Clickhouse server url",
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_HOST")
					},
				},
				"port": {
					Description: "Clickhouse server native protocol port (TCP)",
					Type:        schema.TypeInt,
					Required:    true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_PORT")
					},
				},
				"secure": {
					Description: "Clickhouse secure connection",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"clickhouse_dbs": datasources.DataSourceDbs(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"clickhouse_db":    resourcedb.ResourceDb(),
				"clickhouse_table": resourcetable.ResourceTable(),
				"clickhouse_view":  resourceview.ResourceView(),
				"clickhouse_role":  resourcerole.ResourceRole(),
				"clickhouse_user":  resourceuser.ResourceUser(),
			},
			ConfigureContextFunc: configure(),
		}

		return p
	}
}

func getEnvVar(envVarName string) (any, error) {

	godotenv.Load(".env")
	if v := os.Getenv(envVarName); v != "" {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("Env var %v not present", envVarName))

}

func configure() func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
		host := d.Get("host").(string)
		port := d.Get("port").(int)
		username := d.Get("username").(string)
		defaultCluster := d.Get("default_cluster").(string)
		password := d.Get("password").(string)
		secure := d.Get("secure").(bool)

		// Check if TF_LOG is set to DEBUG or TRACE
		tfLogLevel := strings.ToUpper(os.Getenv("TF_LOG"))
		debugEnabled := tfLogLevel == "DEBUG" || tfLogLevel == "TRACE"
		println(tfLogLevel)
		println(debugEnabled)

		var TLSConfig *tls.Config
		// To use TLS it's necessary to set the TLSConfig field as not nil
		if secure {
			TLSConfig = &tls.Config{
				InsecureSkipVerify: false,
			}
		}
		conn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{fmt.Sprintf("%s:%d", host, port)},
			Auth: clickhouse.Auth{
				Username: username,
				Password: password,
			},
			Debug: debugEnabled,
			Debugf: func(format string, v ...any) {
				if debugEnabled {
					cleanedFormat := strings.ReplaceAll(format, "\t", "    ")
					cleanedArgs := make([]interface{}, len(v))
					for i, arg := range v {
						if str, ok := arg.(string); ok {
							cleanedArgs[i] = strings.ReplaceAll(str, "\t", "    ")
						} else {
							cleanedArgs[i] = arg
						}
					}
					fmt.Printf(cleanedFormat+"\n", cleanedArgs...)
				}
			},
			Settings: clickhouse.Settings{
				"max_execution_time": 300,
			},
			TLS: TLSConfig,
		})

		var diags diag.Diagnostics

		if err != nil {
			return nil, diag.FromErr(fmt.Errorf("error connecting to clickhouse: %v", err))
		}

		if err := conn.Ping(ctx); err != nil {
			return nil, diag.FromErr(fmt.Errorf("ping clickhouse database: %w", err))
		}

		return &common.ApiClient{ClickhouseConnection: &conn, DefaultCluster: defaultCluster}, diags
	}
}
