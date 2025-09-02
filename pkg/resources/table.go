package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/common"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/models"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceTable() *schema.Resource {
	return &schema.Resource{
		Description: "Resource to manage tables",

		CreateContext: resourceTableCreate,
		ReadContext:   resourceTableRead,
		DeleteContext: resourceTableDelete,
		UpdateContext: resourceTableUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), ":")
				switch len(idParts) {
				case 3:
					if err := d.Set("cluster", idParts[0]); err != nil {
						return nil, err
					}
					if err := d.Set("database", idParts[1]); err != nil {
						return nil, err
					}
					if err := d.Set("name", idParts[2]); err != nil {
						return nil, err
					}
				case 2:
					if err := d.Set("database", idParts[0]); err != nil {
						return nil, err
					}
					if err := d.Set("name", idParts[1]); err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("invalid import ID, expected <database>:<table> or <cluster>:<database>:<table>")
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"database": {
				Description: "DB Name where the table will bellow",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"comment": {
				Description: "Database comment, it will be codified in a json along with come metadata information (like cluster name in case of clustering)",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"name": {
				Description: "Table Name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"cluster": {
				Description: "Cluster Name, it is required for Replicated or Distributed tables and forbidden in other case",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"engine": {
				Description: "Table engine type (Supported types so far: Distributed, ReplicatedReplacingMergeTree, ReplacingMergeTree)",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"engine_params": {
				Description: "Engine params in case the engine type requires them",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"primary_key": {
				Description: "Columns to use as primary key",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"order_by": {
				Description: "Order by columns to use as sorting key",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"partition_by": {
				Description: "Partition Key to split data",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"by": {
							Description: "Column to use as part of the partition key",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"partition_function": {
							Description: "Partition function, could be empty or one of following: toYYYYMM, toYYYYMMDD or toYYYYMMDDhhmmss",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     nil,
							ForceNew:    true,
						},
						"mod": {
							Description: "Modulo to apply to the partition function",
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"column": {
				Description: "Column",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Column Name",
							Type:        schema.TypeString,
							Required:    true,
						},
						"type": {
							Description: "Column Type",
							Type:        schema.TypeString,
							Required:    true,
						},
						"comment": {
							Description: "Column Comment",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
						},
						"default_kind": {
							Description: "Column Default Kind",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
						},
						"default_expression": {
							Description: "Column Default Expression",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
						},
						"compression_codec": {
							Description: "Column codec compression",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
						},
					},
				},
			},
			"settings": {
				Description: "Table settings",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ttl": {
				Description: "Table TTL",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"index": {
				Description: "Index",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Index Name",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"expression": {
							Description: "Index Expression",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"type": {
							Description: "Index Type",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"granularity": {
							Description: "Index Granularity",
							Type:        schema.TypeInt,
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
		},
	}
}

func resourceTableRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	c := meta.(*sdk.Client)
	database := d.Get("database").(string)
	tableName := d.Get("name").(string)

	chTable, err := c.GetTable(ctx, database, tableName)
	if chTable == nil && err == nil {
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("reading Clickhouse table: %v", err))
	}

	tableResource, err := chTable.ToResource()
	if err != nil {
		return diag.FromErr(fmt.Errorf("transforming Clickhouse table to resource: %v", err))
	}

	if err := d.Set("database", tableResource.Database); err != nil {
		return diag.FromErr(fmt.Errorf("setting database: %v", err))
	}
	if tableResource.Comment != "" {
		if err := d.Set("comment", tableResource.Comment); err != nil {
			return diag.FromErr(fmt.Errorf("setting comment: %v", err))
		}
	}
	if err := d.Set("name", tableResource.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %v", err))
	}
	if tableResource.Cluster != "" {
		if err := d.Set("cluster", tableResource.Cluster); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster: %v", err))
		}
	}
	if err := d.Set("engine", tableResource.Engine); err != nil {
		return diag.FromErr(fmt.Errorf("setting engine: %v", err))
	}
	if tableResource.EngineParams != nil {
		if err := d.Set("engine_params", tableResource.EngineParams); err != nil {
			return diag.FromErr(fmt.Errorf("setting engine_params: %v", err))
		}
	}
	if tableResource.PrimaryKey != nil {
		if err := d.Set("primary_key", tableResource.OrderBy); err != nil {
			return diag.FromErr(fmt.Errorf("setting order_by: %v", err))
		}
	}
	if tableResource.OrderBy != nil {
		if err := d.Set("order_by", tableResource.OrderBy); err != nil {
			return diag.FromErr(fmt.Errorf("setting order_by: %v", err))
		}
	}
	// not set - partition_by
	if err := d.Set("column", c.GetColumnDefintions(tableResource.Columns)); err != nil {
		return diag.FromErr(fmt.Errorf("setting column: %v", err))
	}
	if tableResource.Indexes != nil {
		if err := d.Set("index", c.GetIndexDefintions(tableResource.Indexes)); err != nil {
			return diag.FromErr(fmt.Errorf("setting indexes: %v", err))
		}
	}
	// not set - settings

	d.SetId(tableResource.Cluster + ":" + database + ":" + tableName)

	return diags
}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	c := meta.(*sdk.Client)
	tableResource := models.TableResource{}

	tableResource.Cluster = d.Get("cluster").(string)
	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.SetColumns(d.Get("column").([]interface{}))
	tableResource.SetIndexes(d.Get("index").([]interface{}))
	tableResource.Engine = d.Get("engine").(string)
	tableResource.Comment = d.Get("comment").(string)
	tableResource.EngineParams = common.MapArrayInterfaceToArrayOfStrings(d.Get("engine_params").([]interface{}))
	tableResource.PrimaryKey = common.MapArrayInterfaceToArrayOfStrings(d.Get("primary_key").([]interface{}))
	tableResource.OrderBy = common.MapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	tableResource.SetPartitionBy(d.Get("partition_by").([]interface{}))
	tableResource.Settings = common.MapInterfaceToMapOfString(d.Get("settings").(map[string]interface{}))
	tableResource.TTL = common.MapInterfaceToMapOfString(d.Get("ttl").(map[string]interface{}))

	tableResource.Validate(diags)
	if diags.HasError() {
		return diags
	}

	err := c.CreateTable(ctx, tableResource)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tableResource.Cluster + ":" + tableResource.Database + ":" + tableResource.Name)

	return diags
}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	c := meta.(*sdk.Client)

	var tableResource models.TableResource
	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.Cluster = d.Get("cluster").(string)

	err := c.DeleteTable(ctx, tableResource)

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceTableUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	c := meta.(*sdk.Client)

	tableResource := models.TableResource{}

	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.Cluster = d.Get("cluster").(string)
	tableResource.SetColumns(d.Get("column").([]interface{}))
	tableResource.Comment = d.Get("comment").(string)
	tableResource.TTL = common.MapInterfaceToMapOfString(d.Get("ttl").(map[string]interface{}))

	err := c.UpdateTable(ctx, tableResource, d)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
