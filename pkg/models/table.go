package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type CHTable struct {
	Database   string     `ch:"database"`
	Name       string     `ch:"name"`
	EngineFull string     `ch:"engine_full"`
	SortingKey string     `ch:"sorting_key"`
	Engine     string     `ch:"engine"`
	Comment    string     `ch:"comment"`
	Columns    []CHColumn `ch:"columns"`
	Indexes    []CHIndex  `ch:"indexes"`
}

type CHIndex struct {
	Name        string `ch:"name"`
	Expression  string `ch:"expr"`
	Type        string `ch:"type"`
	Granularity uint64 `ch:"granularity"`
}

type CHColumn struct {
	Database          string `ch:"database"`
	Table             string `ch:"table"`
	Name              string `ch:"name"`
	Type              string `ch:"type"`
	Comment           string `ch:"comment"`
	DefaultKind       string `ch:"default_kind"`
	DefaultExpression string `ch:"default_expression"`
	CompressionCodec  string `ch:"compression_codec"`
}

type TableResource struct {
	Database     string
	Name         string
	EngineFull   string
	Engine       string
	Cluster      string
	Comment      string
	EngineParams []string
	PrimaryKey   []string
	OrderBy      []string
	Columns      []ColumnDefinition
	PartitionBy  []PartitionByResource
	Indexes      []IndexDefinition
	Settings     map[string]string
	TTL          map[string]string
}

type IndexDefinition struct {
	Name        string
	Expression  string
	Type        string
	Granularity uint64
}

type ColumnDefinition struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Comment           string `json:"comment"`
	DefaultKind       string `json:"default_kind"`
	DefaultExpression string `json:"default_expression"`
	CompressionCodec  string `json:"compression_codec"`
}

type PartitionByResource struct {
	By                string
	PartitionFunction string
	Mod               string
}

func (t *CHTable) IndexesToResource() []IndexDefinition {
	indexResources := make([]IndexDefinition, len(t.Indexes))
	for i, index := range t.Indexes {
		indexResources[i] = IndexDefinition(index)
	}
	return indexResources
}

func (t *CHTable) ColumnsToResource() []ColumnDefinition {
	var columnResources []ColumnDefinition
	for _, column := range t.Columns {
		columnResource := ColumnDefinition{
			Name:              column.Name,
			Type:              column.Type,
			Comment:           column.Comment,
			DefaultKind:       column.DefaultKind,
			DefaultExpression: column.DefaultExpression,
			CompressionCodec:  column.CompressionCodec,
		}
		columnResources = append(columnResources, columnResource)
	}
	return columnResources
}

func (t *CHTable) ToResource() (*TableResource, error) {
	tableResource := TableResource{
		Database:     t.Database,
		Name:         t.Name,
		EngineFull:   t.EngineFull,
		Engine:       t.Engine,
		EngineParams: removeDefaultParams(GetEngineParams(t.EngineFull)),
		OrderBy:      GetOrderBy(t.SortingKey),
		Columns:      t.ColumnsToResource(),
		Indexes:      t.IndexesToResource(),
		Comment:      t.Comment,
	}

	return &tableResource, nil
}

func GetEngineParams(engineFull string) []string {
	r := regexp.MustCompile(`^\w+\(([^)]*)\)`)
	match := r.FindStringSubmatch(engineFull)
	var engineParams []string
	if len(match) > 1 {
		values := strings.Split(match[1], ",")
		for _, value := range values {
			value = strings.TrimSpace(value)
			engineParams = append(engineParams, strings.TrimSpace(value))
		}
	}
	return engineParams
}

func GetOrderBy(sortingKey string) []string {
	var orderBy []string
	sortingKey = strings.TrimSpace(sortingKey)

	if sortingKey == "tuple()" || sortingKey == "" {
		return orderBy
	}

	keys := strings.Split(sortingKey, ",")

	for _, key := range keys {
		key = strings.TrimSpace(key)
		orderBy = append(orderBy, key)
	}
	return orderBy
}

// without this, terraform sees a diff for Replicated tables
func removeDefaultParams(engineParams []string) []string {
	var newEngineParams []string
	for _, param := range engineParams {
		if param != "'/clickhouse/tables/{uuid}/{shard}'" && param != "'{replica}'" {
			newEngineParams = append(newEngineParams, param)
		}
	}
	return newEngineParams
}

func (t *TableResource) GetColumnsResourceList() []ColumnDefinition {
	var columnResources []ColumnDefinition
	for _, column := range t.Columns {
		columnResources = append(columnResources, ColumnDefinition{
			Name:              column.Name,
			Type:              column.Type,
			Comment:           column.Comment,
			DefaultKind:       column.DefaultKind,
			DefaultExpression: column.DefaultExpression,
			CompressionCodec:  column.CompressionCodec,
		})
	}
	return columnResources
}

func (t *TableResource) SetPartitionBy(partitionBy []interface{}) {
	for _, partitionBy := range partitionBy {
		partitionByResource := PartitionByResource{
			By:                partitionBy.(map[string]interface{})["by"].(string),
			PartitionFunction: partitionBy.(map[string]interface{})["partition_function"].(string),
			Mod:               partitionBy.(map[string]interface{})["mod"].(string),
		}
		t.PartitionBy = append(t.PartitionBy, partitionByResource)
	}
}

func (t *TableResource) HasColumn(columnName string) bool {
	for _, column := range t.GetColumnsResourceList() {
		if column.Name == columnName {
			return true
		}
	}
	return false
}

func (t *TableResource) Validate(diags diag.Diagnostics) {
	t.validateOrderBy(diags)
	t.validatePartitionBy(diags)
}

func (t *TableResource) validateOrderBy(diags diag.Diagnostics) {
	for _, orderField := range t.OrderBy {
		if !t.HasColumn(orderField) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("order by field '%s' is not a column", orderField),
			})
		}
	}
}

func (t *TableResource) validatePartitionBy(diags diag.Diagnostics) {
	for _, partitionBy := range t.PartitionBy {
		if !t.HasColumn(partitionBy.By) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("partition by field '%s' is not a column", partitionBy.By),
			})
		}
	}
}

func (t *TableResource) SetColumns(columns []interface{}) {
	for _, column := range columns {
		columnDefinition := ColumnDefinition{
			Name:              column.(map[string]interface{})["name"].(string),
			Type:              column.(map[string]interface{})["type"].(string),
			Comment:           column.(map[string]interface{})["comment"].(string),
			DefaultKind:       column.(map[string]interface{})["default_kind"].(string),
			DefaultExpression: column.(map[string]interface{})["default_expression"].(string),
			CompressionCodec:  column.(map[string]interface{})["compression_codec"].(string),
		}
		t.Columns = append(t.Columns, columnDefinition)
	}
}

func (t *TableResource) SetIndexes(indexes []interface{}) {
	for _, index := range indexes {
		indexDefinition := IndexDefinition{
			Name:        index.(map[string]interface{})["name"].(string),
			Expression:  index.(map[string]interface{})["expression"].(string),
			Type:        index.(map[string]interface{})["type"].(string),
			Granularity: uint64(index.(map[string]interface{})["granularity"].(int)),
		}
		t.Indexes = append(t.Indexes, indexDefinition)
	}
}
