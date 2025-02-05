package queryparams

import (
	"fmt"
	"montelukast/modules/product/entity"
	"montelukast/pkg/dateconverter"
	"strings"
)

type QueryParams struct {
	Name       string
	CategoryID int
	Limit      int
	Page       int
	SortBy     string
	Order      string
}

type QueryParamsDto struct {
	Name       string `form:"name"`
	CategoryID int    `form:"category_id"`
	SortBy     string `form:"sort_by"`
	Order      string `form:"order"`
	Limit      int    `form:"limit"`
	Page       int    `form:"page"`
}

func AddConditionQuery(params *[]any, queryParams QueryParams, querIndex *int) string {
	var query string

	if queryParams.Name != "" {
		queryParams.Name = "%" + queryParams.Name + "%"
		query += fmt.Sprintf(" AND product_name ILIKE $%d", *querIndex)
		*querIndex++
		*params = append(*params, queryParams.Name)
	}

	query += fmt.Sprintf(" AND start_hour < $%d", *querIndex)
	query += fmt.Sprintf(" AND end_hour > $%d", *querIndex)
	*params = append(*params, dateconverter.GetCurrentTime())	
	*querIndex++

	query += fmt.Sprintf(" AND active_days ilike $%d", *querIndex)
	*params = append(*params, "%" + dateconverter.GetCurrentDay() + "%")
	*querIndex++

	return query
}

func AddPaginationQuery(params *[]any, queryParams QueryParams, querIndex *int) string {
	query := ``
	if queryParams.Limit != 0 {
		query += fmt.Sprintf(" LIMIT $%d", *querIndex)
		*querIndex++
		*params = append(*params, queryParams.Limit)
	}

	if queryParams.Page != 0 {
		query += fmt.Sprintf(" OFFSET %d", (queryParams.Page-1)*queryParams.Limit)
	}

	return query
}

func AddFilterByCategoryID(params *[]any, queryParams QueryParams, querIndex *int, categoryBoundary entity.CategoryBoundary) string {
	query := `WITH ProductCategory as ( 
					SELECT p.id as product_id, p.name as product_name, p.image[1] as image, p.manufacture as manufacture
					FROM products p),`
	if queryParams.CategoryID >= categoryBoundary.Minimum && queryParams.CategoryID <= categoryBoundary.Maximum {
		query = strings.TrimSuffix(query, "),")
		query += `  JOIN product_multi_categories pmc on pmc.product_id = p.id
					JOIN product_categories pc on pc.id = pmc.product_category_id
					WHERE p.deleted_at IS null and pc.id = $1),`
		*querIndex++
		*params = append(*params, queryParams.CategoryID)
	}
	return query
} 
