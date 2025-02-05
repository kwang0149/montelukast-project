package repository

import (
	"context"
	"database/sql"
	"fmt"

	"montelukast/modules/product/entity"
	queryparams "montelukast/modules/product/queryparams"

	appconstant "montelukast/pkg/constant"

	apperror "montelukast/pkg/error"
	"montelukast/pkg/transaction"
)

type ProductRepo interface {
	GetUserProducts(c context.Context, queryParams queryparams.QueryParams, location string, categoryBoundary entity.CategoryBoundary) ([]entity.ProductDetail, error)
	GetTotalProduct(c context.Context, queryParams queryparams.QueryParams, location string, categoryBoundary entity.CategoryBoundary) (int, error)
	IsAddressExistsByUserID(c context.Context, userID int) (bool, error)
	GetAddressByUserID(c context.Context, userID int) (int, error)
	GetProductDetail(c context.Context, pharcistsProductID int) (*entity.ProductDetail, error)
	GetProductsAdmin(c context.Context, queryParams queryparams.AdminQueryParams) ([]entity.ProductAdmin, error)
	GetTotalProductAdmin(c context.Context, queryParams queryparams.AdminQueryParams) (int, error)
	IsProductExistsByID(c context.Context, productID int) (bool, error)
	UpdateProduct(c context.Context, product entity.Product) error
	DeleteProduct(c context.Context, productID int) error
	AddMultipleCategories(c context.Context, product entity.Product) error
	DeleteMultiCategories(c context.Context, productID int) error
	GetProductCategories(c context.Context, productID int) ([]string, error)
	GetLocationByAddressID(c context.Context, addressID int) (string, error)
	GetTotalProductHomePage(c context.Context, queryParams queryparams.QueryParams, location string) (int, error)
	GetUserProductsHomePage(c context.Context, queryParams queryparams.QueryParams, location string) ([]entity.ProductDetail, error)
	IsPharmacyProductExistsByID(c context.Context, id int) (bool, error)
	GetMasterProducts(c context.Context, queryParams queryparams.QueryParams) ([]entity.ProductDetail, error)
	GetTotalMasterProduct(c context.Context, queryParams queryparams.QueryParams) (int, error)
	AddProduct(c context.Context, product *entity.Product) error
	IsProductExists(c context.Context, product entity.Product) (bool, error)
	DeletePharmacyDeletedProducts(c context.Context, productID int) error
	UpdateProductPhoto(c context.Context, url string, productID int) error
	GetCategoryBoundary(c context.Context) (*entity.CategoryBoundary, error)
	GetTotalProductCategories(c context.Context, categories []int) (int, error)
}

type ProductRepoImpl struct {
	db *sql.DB
}

func NewProductRepo(dbConn *sql.DB) ProductRepoImpl {
	return ProductRepoImpl{
		db: dbConn,
	}
}

func (r ProductRepoImpl) GetUserProducts(c context.Context, queryParams queryparams.QueryParams, location string, categoryBoundary entity.CategoryBoundary) ([]entity.ProductDetail, error) {
	products := []entity.ProductDetail{}
	var params []any
	querIndex := 1

	query := queryparams.AddFilterByCategoryID(&params, queryParams, &querIndex, categoryBoundary)
	query += fmt.Sprintf(` GetDistance AS (
					SELECT p.product_id, pp.id as pharmacy_product_id, p.image, p.product_name, p.manufacture, ph.name as pharmacy_product_name, pp.price as product_price, st_distance(ph.location, $%d::geography) as distance
					FROM ProductCategory p
					JOIN pharmacy_products pp ON pp.product_id = p.product_id AND pp.stock > 0 AND pp.deleted_at IS NULL
					JOIN pharmacies ph ON ph.id = pp.pharmacy_id AND pp.is_active = true AND ph.deleted_at IS NULL
					WHERE ST_DWithin(ph.location, $%d::geography, 25000)
				), DetermineProductRank AS (
					SELECT product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, rank() over (order by product_price desc) * 0.7 as price_score, rank() over (order by distance desc) * 0.3 as distance_score
					FROM GetDistance
				)
				SELECT DISTINCT ON (dpr.product_id) dpr.product_id, dpr.pharmacy_product_id, dpr.image, dpr.product_name, dpr.manufacture, dpr.pharmacy_product_name, dpr.product_price, price_score + distance_score as total_score
				FROM DetermineProductRank dpr
				join pharmacy_products pp on pp.id = dpr.pharmacy_product_id AND pp.is_active = true AND pp.deleted_at IS NULL
				join pharmacies ph on ph.id = pp.pharmacy_id AND ph.is_active = true AND ph.deleted_at IS NULL
				join products p on p.id = dpr.product_id AND p.is_active = true AND p.deleted_at IS NULL
				join partners pt on pt.id = ph.partner_id AND pt.is_active = true AND pt.deleted_at IS NULL
				where 1=1`, querIndex, querIndex)

	params = append(params, location)
	querIndex++

	query += queryparams.AddConditionQuery(&params, queryParams, &querIndex)
	query += ` ORDER BY product_id, total_score DESC`
	query += queryparams.AddPaginationQuery(&params, queryParams, &querIndex)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	defer rows.Close()

	var score float64
	for rows.Next() {
		var product entity.ProductDetail
		err := rows.Scan(
			&product.ID,
			&product.PharmacyProductID,
			&product.Image,
			&product.Name,
			&product.Manufacture,
			&product.PharmacyName,
			&product.Price,
			&score,
		)
		if err != nil {
			return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
		}
		products = append(products, product)
	}
	return products, nil
}

func (r ProductRepoImpl) GetTotalProduct(c context.Context, queryParams queryparams.QueryParams, location string, categoryBoundary entity.CategoryBoundary) (int, error) {
	queryParams.Limit = 0
	queryParams.Page = 0
	queryParams.SortBy = ""
	queryParams.Order = ""

	var params []any
	querIndex := 1

	query := queryparams.AddFilterByCategoryID(&params, queryParams, &querIndex, categoryBoundary)
	query += fmt.Sprintf(` GetDistance AS (
					SELECT p.product_id, pp.id as pharmacy_product_id, p.image, p.product_name, p.manufacture, ph.name as pharmacy_product_name, pp.price as product_price, st_distance(ph.location, $%d::geography) as distance
					FROM ProductCategory p
					JOIN pharmacy_products pp ON pp.product_id = p.product_id AND pp.stock > 0 AND pp.deleted_at IS NULL
					JOIN pharmacies ph ON ph.id = pp.pharmacy_id AND pp.is_active = true AND ph.deleted_at IS NULL
					WHERE ST_DWithin(ph.location, $%d::geography, 25000)
				), DetermineProductRank AS (
					SELECT product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, rank() over (order by product_price desc) * 0.7 as price_score, rank() over (order by distance desc) * 0.3 as distance_score
					FROM GetDistance
				)
				SELECT COUNT(DISTINCT dpr.product_id)
				FROM DetermineProductRank dpr
				join pharmacy_products pp on pp.id = dpr.pharmacy_product_id AND pp.is_active = true AND pp.deleted_at IS NULL
				join pharmacies ph on ph.id = pp.pharmacy_id AND ph.is_active = true AND ph.deleted_at IS NULL
				join products p on p.id = dpr.product_id AND p.is_active = true AND p.deleted_at IS NULL
				join partners pt on pt.id = ph.partner_id AND pt.is_active = true AND pt.deleted_at IS NULL
				where 1=1`, querIndex, querIndex)

	params = append(params, location)
	querIndex++
	query += queryparams.AddConditionQuery(&params, queryParams, &querIndex)
	query += queryparams.AddPaginationQuery(&params, queryParams, &querIndex)

	var totalProduct int
	err := r.db.QueryRow(query, params...).Scan(&totalProduct)
	if err != nil {
		return 0, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return totalProduct, nil
}

func (r ProductRepoImpl) GetTotalProductHomePage(c context.Context, queryParams queryparams.QueryParams, location string) (int, error) {
	queryParams.Limit = 0
	queryParams.Page = 0
	queryParams.SortBy = ""
	queryParams.Order = ""

	var params []any
	querIndex := 1

	query := `with GetDistance as (
					select p.id as product_id, pp.id as pharmacy_product_id, p.image[1] as image, p.name as product_name, p.manufacture as manufacture, ph.name as pharmacy_product_name, pp.price as product_price, st_distance(ph.location, $1::geography) as distance
					FROM products p
					JOIN pharmacy_products pp ON pp.product_id = p.id AND pp.stock > 0 AND pp.deleted_at IS NULL
					JOIN pharmacies ph ON ph.id = pp.pharmacy_id AND pp.is_active = true AND ph.deleted_at IS NULL
					WHERE ST_DWithin(ph.location, $1::geography, 25000) AND p.deleted_at IS NULL
				), DetermineProductRank as (
					select product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, rank() over (order by product_price desc) * 0.7 as price_score, rank() over (order by distance desc) * 0.3 as distance_score
					from GetDistance
				), MostBoughtProduct as (
					select p.id, count(*) as counter
					from order_product_details opd
					right join pharmacy_products pp on pp.id = opd.pharmacy_product_id
					join products p on p.id = pp.product_id
					group by p.id
					order by counter desc
				), NearestCheapestProduct as (
					select distinct on (product_id) product_id as product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, price_score + distance_score as total_score
					from DetermineProductRank dpr
					order by product_id, total_score
				)
				select COUNT(DISTINCT ncp.product_id)
				from NearestCheapestProduct ncp
				join MostBoughtProduct mbp on mbp.id = ncp.product_id
				join pharmacy_products pp on pp.id = ncp.pharmacy_product_id AND pp.is_active = true AND pp.deleted_at IS NULL
				join pharmacies ph on ph.id = pp.pharmacy_id AND ph.is_active = true AND ph.deleted_at IS NULL
				join products p on p.id = ncp.product_id AND p.is_active = true AND p.deleted_at IS NULL
				join partners pt on pt.id = ph.partner_id AND pt.is_active = true AND pt.deleted_at IS NULL
				WHERE 1=1`

	querIndex++

	params = append(params, location)
	query += queryparams.AddConditionQuery(&params, queryParams, &querIndex)
	query += queryparams.AddPaginationQuery(&params, queryParams, &querIndex)

	var totalProduct int
	err := r.db.QueryRow(query, params...).Scan(&totalProduct)
	if err != nil {
		return 0, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return totalProduct, nil
}

func (r ProductRepoImpl) GetUserProductsHomePage(c context.Context, queryParams queryparams.QueryParams, location string) ([]entity.ProductDetail, error) {
	products := []entity.ProductDetail{}

	query := `with GetDistance as (
					select p.id as product_id, pp.id as pharmacy_product_id, p.image[1] as image, p.name as product_name, p.manufacture as manufacture, ph.name as pharmacy_product_name, pp.price as product_price, st_distance(ph.location, $1::geography) as distance
					FROM products p
					JOIN pharmacy_products pp ON pp.product_id = p.id AND pp.stock > 0 AND pp.deleted_at IS NULL
					JOIN pharmacies ph ON ph.id = pp.pharmacy_id AND pp.is_active = true AND ph.deleted_at IS NULL
					WHERE ST_DWithin(ph.location, $1::geography, 25000) AND p.deleted_at IS NULL  
				), DetermineProductRank as (
					select product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, rank() over (order by product_price desc) * 0.7 as price_score, rank() over (order by distance desc) * 0.3 as distance_score
					from GetDistance
				), MostBoughtProduct as (
					select p.id, count(*) as counter
					from order_product_details opd
					right join pharmacy_products pp on pp.id = opd.pharmacy_product_id
					join products p on p.id = pp.product_id
					group by p.id
					order by counter desc
				), NearestCheapestProduct as (
					select distinct on (product_id) product_id as product_id, pharmacy_product_id, image, product_name, manufacture, pharmacy_product_name, product_price, distance, price_score + distance_score as total_score
					from DetermineProductRank dpr
					order by product_id, total_score
				)
				select ncp.product_id, pharmacy_product_id, ncp.image, product_name, ncp.manufacture, pharmacy_product_name, product_price, counter
				from NearestCheapestProduct ncp
				join MostBoughtProduct mbp on mbp.id = ncp.product_id
				join pharmacy_products pp on pp.id = ncp.pharmacy_product_id AND pp.is_active = true AND pp.deleted_at IS NULL
				join pharmacies ph on ph.id = pp.pharmacy_id AND ph.is_active = true AND pp.deleted_at IS NULL
				join products p on p.id = ncp.product_id AND p.is_active = true AND pp.deleted_at IS NULL
				join partners pt on pt.id = ph.partner_id AND pt.is_active = true AND pt.deleted_at IS NULL
				WHERE 1=1`

	var params []any
	querIndex := 1
	querIndex++

	params = append(params, location)
	query += queryparams.AddConditionQuery(&params, queryParams, &querIndex)
	query += ` order by counter desc`
	query += queryparams.AddPaginationQuery(&params, queryParams, &querIndex)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	defer rows.Close()

	var score float64
	for rows.Next() {
		var product entity.ProductDetail
		err := rows.Scan(
			&product.ID,
			&product.PharmacyProductID,
			&product.Image,
			&product.Name,
			&product.Manufacture,
			&product.PharmacyName,
			&product.Price,
			&score,
		)
		if err != nil {

			return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
		}
		products = append(products, product)
	}
	return products, nil
}

func (r ProductRepoImpl) IsAddressExistsByUserID(c context.Context, userID int) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM user_addresses WHERE user_id = $1 AND is_active = true AND deleted_at IS NULL) FOR UPDATE `

	var exists bool
	err := r.db.QueryRow(query, userID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return exists, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	return exists, nil
}

func (r ProductRepoImpl) GetAddressByUserID(c context.Context, userID int) (int, error) {
	query := `SELECT id
				FROM user_addresses
				WHERE user_id = $1 AND is_active = true AND deleted_at IS NULL`

	var addressID int
	err := r.db.QueryRow(query, userID).Scan(&addressID)
	if err != nil {
		return 0, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return addressID, nil
}

func (r ProductRepoImpl) GetProductCategories(c context.Context, productID int) ([]string, error) {
	categories := []string{}

	query := `select pc.name
				from products p 
				join product_multi_categories pmc on pmc.product_id = p.id 
				join product_categories pc on pc.id = pmc.product_category_id 
				where p.id = $1
				`

	rows, err := r.db.Query(query, productID)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		err := rows.Scan(
			&category,
		)
		if err != nil {
			return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r ProductRepoImpl) GetProductDetail(c context.Context, pharcistsProductID int) (*entity.ProductDetail, error) {
	query := `select p.id, pp.id, p.name, p.image[1], p.generic_name, p.manufacture, p.description, p.unit_in_pack, ph.name, ph.address, pp.stock, pp.price 
				from pharmacy_products pp 
				join products p on p.id = pp.product_id and p.deleted_at is null
				join pharmacies ph on ph.id = pp.pharmacy_id and ph.deleted_at is null
				where pp.id = $1 and p.deleted_at is null`

	var productDetail entity.ProductDetail
	err := r.db.QueryRow(query, pharcistsProductID).Scan(
		&productDetail.ID,
		&productDetail.PharmacyProductID,
		&productDetail.Name,
		&productDetail.Image,
		&productDetail.GenericName,
		&productDetail.Manufacture,
		&productDetail.Description,
		&productDetail.UnitInPack,
		&productDetail.PharmacyName,
		&productDetail.PharmacyAddress,
		&productDetail.Stock,
		&productDetail.Price,
	)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return &productDetail, nil
}

func (r ProductRepoImpl) AddProduct(c context.Context, product *entity.Product) error {
	tx := transaction.ExtractTx(c)

	query := `INSERT INTO products (product_classification_id, product_form_id, name, generic_name, manufacture, description, image, unit_in_pack, weight, height, length, width, is_active)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			  RETURNING id`

	var err error

	if tx != nil {
		err = tx.QueryRowContext(c, query,
			product.ProductClassificationID, product.ProductFormID,
			product.Name, product.GenericName, product.Manufacture, product.Description,
			[]string{product.Image}, product.UnitInPack, product.Weight, product.Height, product.Length, product.Width,
			product.IsActive,
		).Scan(&product.ID)
	} else {
		err = r.db.QueryRowContext(c, query,
			product.ProductClassificationID, product.ProductFormID,
			product.Name, product.GenericName, product.Manufacture, product.Description,
			[]string{product.Image}, product.UnitInPack, product.Weight, product.Height, product.Length, product.Width,
			product.IsActive,
		).Scan(&product.ID)
	}

	if err != nil {
		return apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	return nil
}

func (r ProductRepoImpl) IsProductExists(c context.Context, product entity.Product) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM products WHERE name = $1 AND generic_name = $2 AND manufacture = $3 AND deleted_at IS NULL) FOR UPDATE `

	var exists bool
	err := r.db.QueryRow(query, product.Name, product.GenericName, product.Manufacture).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return exists, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	return exists, nil
}

func (r ProductRepoImpl) IsProductExistsByID(c context.Context, productID int) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1 AND deleted_at IS NULL) FOR UPDATE `

	var exists bool
	err := r.db.QueryRow(query, productID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return exists, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	return exists, nil
}

func (r ProductRepoImpl) GetLocationByAddressID(c context.Context, addressID int) (string, error) {
	query := `SELECT location
				FROM user_addresses
				WHERE id = $1 AND is_active = true AND deleted_at IS NULL`

	var location string
	err := r.db.QueryRow(query, addressID).Scan(&location)
	if err != nil {
		return "", apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return location, nil
}

func (r ProductRepoImpl) IsPharmacyProductExistsByID(c context.Context, id int) (bool, error) {
	query := `SELECT EXISTS (
							SELECT 1 
							FROM pharmacy_products pp 
							JOIN products p 
							ON pp.product_id = p.id 
							WHERE pp.id = $1 
							AND p.deleted_at IS NULL 
							AND pp.deleted_at IS NULL)`

	var isExists bool
	err := r.db.QueryRowContext(c, query, id).Scan(&isExists)
	if err != nil && err != sql.ErrNoRows {
		return isExists, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	return isExists, nil
}

func (r ProductRepoImpl) GetMasterProducts(c context.Context, queryParams queryparams.QueryParams) ([]entity.ProductDetail, error) {
	products := []entity.ProductDetail{}
	querIndex := 1

	query := `SELECT p.id, p.name
				FROM products p 
				WHERE p.is_active = True AND p.deleted_at IS NULL`

	var params []any

	query += queryparams.AddMasterConditionQuery(&params, queryParams, &querIndex)
	query += queryparams.AddMasterSortByQuery(queryParams)
	query += queryparams.AddMasterPaginationQuery(&params, queryParams, &querIndex)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}
	defer rows.Close()

	for rows.Next() {
		var product entity.ProductDetail
		err := rows.Scan(
			&product.ID,
			&product.Name,
		)
		if err != nil {
			return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
		}
		products = append(products, product)
	}

	return products, nil
}

func (r ProductRepoImpl) GetTotalMasterProduct(c context.Context, queryParams queryparams.QueryParams) (int, error) {
	queryParams.Limit = 0
	queryParams.Page = 0
	queryParams.SortBy = ""
	queryParams.Order = ""
	querIndex := 1

	query := `SELECT COUNT(*)
				FROM products p 
				WHERE p.is_active = True AND p.deleted_at IS NULL`

	var params []any

	query += queryparams.AddMasterConditionQuery(&params, queryParams, &querIndex)
	query += queryparams.AddMasterPaginationQuery(&params, queryParams, &querIndex)

	var totalProduct int
	err := r.db.QueryRow(query, params...).Scan(&totalProduct)
	if err != nil {
		return 0, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return totalProduct, nil
}

func (r ProductRepoImpl) GetCategoryBoundary(c context.Context) (*entity.CategoryBoundary, error) {
	categoryBoundary := entity.CategoryBoundary{}

	query := `SELECT MAX(id), MIN(id)
				FROM product_categories pc 
				WHERE deleted_at IS NULL`

	err := r.db.QueryRow(query).Scan(&categoryBoundary.Maximum, &categoryBoundary.Minimum)
	if err != nil {
		return nil, apperror.NewErrInternalServerError(appconstant.FieldErrServer, apperror.ErrInternalServer, err)
	}

	return &categoryBoundary, nil
}
