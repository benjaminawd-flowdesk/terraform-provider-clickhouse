package resources

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/common"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/models"
	"github.com/FlowdeskMarkets/terraform-provider-clickhouse/pkg/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceView() *schema.Resource {
	return &schema.Resource{
		Description: "Resource to manage views",

		CreateContext: resourceViewCreate,
		ReadContext:   resourceViewRead,
		DeleteContext: resourceViewDelete,
		Schema: map[string]*schema.Schema{
			"database": {
				Description: "DB Name where the view will bellow",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"comment": {
				Description: "View comment, it will be codified in a json along with come metadata information (like cluster name in case of clustering)",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"name": {
				Description: "View Name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"cluster": {
				Description: "Cluster Name",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			"query": {
				Description: "View query",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				StateFunc: func(val interface{}) string {
					return common.NormalizeQuery(val.(string))
				},
			},
			"materialized": {
				Description: "Is materialized view",
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
			},
			"to_table": {
				Description: "For materialized view - destination table",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
		},
	}
}

func resourceViewRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	writer := bufio.NewWriter(os.Stdout)

	defer func() {
		if err := writer.Flush(); err != nil {
			fmt.Printf("Error flushing writer: %v", err)
		}
	}()

	var diags diag.Diagnostics

	c := meta.(*sdk.Client)
	database := d.Get("database").(string)
	viewName := d.Get("name").(string)

	chView, err := c.GetView(ctx, database, viewName)

	if chView == nil && err == nil {
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("reading Clickhouse view: %v", err))
	}

	viewResource, err := chView.ToResource()
	if err != nil {
		return diag.FromErr(fmt.Errorf("transforming Clickhouse view to resource: %v", err))
	}

	if err := d.Set("database", viewResource.Database); err != nil {
		return diag.FromErr(fmt.Errorf("setting database: %v", err))
	}
	if err := d.Set("name", viewResource.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %v", err))
	}

	if viewResource.Cluster != "" {
		if err := d.Set("cluster", viewResource.Cluster); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster: %v", err))
		}
	}
	if err := d.Set("query", viewResource.Query); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster: %v", err))
	}
	if err := d.Set("materialized", viewResource.Materialized); err != nil {
		return diag.FromErr(fmt.Errorf("setting materialized: %v", err))
	}
	if viewResource.ToTable != "" {
		if err := d.Set("to_table", viewResource.ToTable); err != nil {
			return diag.FromErr(fmt.Errorf("setting to_table: %v", err))
		}
	}

	d.SetId(viewResource.Cluster + ":" + database + ":" + viewName)

	return diags
}

func resourceViewCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := meta.(*sdk.Client)
	viewResource := models.ViewResource{}

	viewResource.Cluster = d.Get("cluster").(string)
	viewResource.Database = d.Get("database").(string)
	viewResource.Name = d.Get("name").(string)
	viewResource.Query = d.Get("query").(string)
	viewResource.Materialized = d.Get("materialized").(bool)
	viewResource.ToTable = d.Get("to_table").(string)
	viewResource.Comment = d.Get("comment").(string)

	diags := viewResource.Validate()
	if diags.HasError() {
		return diags
	}

	err := c.CreateView(ctx, viewResource)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(viewResource.Cluster + ":" + viewResource.Database + ":" + viewResource.Name)

	return diags
}

func resourceViewDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	c := meta.(*sdk.Client)

	var viewResource models.ViewResource
	viewResource.Database = d.Get("database").(string)
	viewResource.Name = d.Get("name").(string)
	viewResource.Cluster = d.Get("cluster").(string)

	err := c.DeleteView(ctx, viewResource)

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
