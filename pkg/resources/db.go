package resources

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/sdk"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceDb() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource to handle clickhouse databases.",

		CreateContext: resourceDbCreate,
		ReadContext:   resourceDbRead,
		DeleteContext: resourceDbDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), ":")
				switch len(idParts) {
				case 2:
					if err := d.Set("cluster", idParts[0]); err != nil {
						return nil, err
					}
					if err := d.Set("name", idParts[1]); err != nil {
						return nil, err
					}
				case 1:
					if err := d.Set("name", idParts[0]); err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("invalid import ID, expected <database> or <cluster>:<database>")
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"cluster": {
				Description: "Cluster name, not mandatory but should be provided if creating a db in a clustered server",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "",
			},
			"name": {
				Description: "Database name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"engine": {
				Description: "Database engine",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_path": {
				Description: "Database internal path",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"metadata_path": {
				Description: "Database internal metadata path",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"uuid": {
				Description: "Database UUID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"comment": {
				Description: "Comment about the database",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "",
			},
		},
	}
}

func resourceDbRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := meta.(*sdk.Client)
	var diags diag.Diagnostics
	cluster := d.Get("cluster").(string)

	database_name := d.Get("name").(string)
	row := c.Conn.QueryRow(ctx, fmt.Sprintf("SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases where name = '%v'", database_name))

	if row.Err() != nil {
		return diag.FromErr(fmt.Errorf("reading database from Clickhouse: %v", row.Err()))
	}

	var name, engine, dataPath, metadataPath, uuid, comment string

	err := row.Scan(&name, &engine, &dataPath, &metadataPath, &uuid, &comment)
	if err != nil {
		// If no rows were returned, treat this as a "new" resource that needs to be created
		if err == sql.ErrNoRows {
			d.SetId("")
			return diags
		}
		return diag.FromErr(fmt.Errorf("scanning Clickhouse DB row: %v", err))
	}

	if name == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Database %v not found", database_name),
			Detail:   "Not possible to retrieve db from server. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	err = d.Set("name", name)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set name for db %q", name),
		})
	}
	err = d.Set("engine", engine)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set engine for db %q", name),
		})
	}
	err = d.Set("data_path", dataPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set data_path for db %q", name),
		})
	}
	err = d.Set("metadata_path", metadataPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set metadata_path for db %q", name),
		})
	}
	err = d.Set("uuid", uuid)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set uuid for db %q", name),
		})
	}

	if err := d.Set("comment", comment); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set comment for db %q", name),
		})
	}

	err = d.Set("cluster", cluster)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set cluster for db %q", name),
		})
	}

	d.SetId(cluster + ":" + database_name)

	tflog.Trace(ctx, "DB resource created.")

	return diags
}

func resourceDbCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := meta.(*sdk.Client)
	var diags diag.Diagnostics

	cluster, _ := d.Get("cluster").(string)
	clusterStatement := common.GetClusterStatement(cluster)
	databaseName := d.Get("name").(string)
	comment := d.Get("comment").(string)
	createStatement := common.GetCreateStatement("database")

	query := fmt.Sprintf("%s %v %v COMMENT '%v'", createStatement, databaseName, clusterStatement, comment)
	err := c.Conn.Exec(ctx, query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cluster + ":" + databaseName)

	return diags
}

func resourceDbDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := meta.(*sdk.Client)
	var diags diag.Diagnostics

	databaseName := d.Get("name").(string)

	if databaseName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Database name not found",
			Detail:   "Not possible to destroy resource as the database name was not retrieved succesfully. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	tables, err := c.GetDBTables(ctx, databaseName)

	if err != nil {
		return diag.FromErr(fmt.Errorf("resource db delete: %v", err))
	}
	if len(tables) > 0 {
		var tableNames []string
		for _, table := range tables {
			tableNames = append(tableNames, table.Name)
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to delete db resource %q", databaseName),
			Detail:   fmt.Sprintf("DB resource is used by another resources and is not possible to delete it. Tables: %v.", tableNames),
		})
		return diags
	}

	cluster, _ := d.Get("cluster").(string)
	clusterStatement := common.GetClusterStatement(cluster)

	query := fmt.Sprintf("DROP DATABASE %v %v SYNC", databaseName, clusterStatement)

	err = c.Conn.Exec(ctx, query)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
