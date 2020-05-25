package main

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/graphql-go/graphql"
	"github.com/jmoiron/sqlx"
)

// Resolver struct for storing required data
type Resolver struct {
	userID int
	db *sqlx.DB
}

// LocationsResolver gets locations from database by specified arguments
func (r *Resolver) LocationsResolver(p graphql.ResolveParams) (interface{}, error) {
	name, ok := p.Args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to convert argument to string")
	}

	query := sq.Select("public_id, name, address, created_at, updated_at").From("locations").Where(sq.And{
		sq.Eq{"created_by": r.userID},
		sq.Like{"name": fmt.Sprint("%", name, "%")},
	})

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	locations := []Location{}
	if err := r.db.Select(&locations, queryString, queryStringArgs...); err != nil {
		return nil, err
	}

	return locations, nil
}

// ReceiptsResolver gets receipts from database by specified arguments
func (r *Resolver) ReceiptsResolver(p graphql.ResolveParams) (interface{}, error) {

	query := sq.Select("receipts.public_id, locations.public_id AS location_id, users.public_id AS created_by, locations.name AS name, locations.address AS address, receipts.created_at, receipts.updated_at, ROUND(SUM(items.price * items_in_receipt.amount), 2) AS total_price").From("receipts").Join("locations ON locations.id = receipts.location_id").Join("users ON users.id = receipts.created_by").LeftJoin("items_in_receipt ON items_in_receipt.receipt_id = receipts.id").LeftJoin("items ON items.id = items_in_receipt.item_id").GroupBy("receipts.id").Where(sq.Eq{"receipts.created_by": r.userID})

	source, sourceOk := p.Source.(Location)
	locationID, locationIDOk := p.Args["locationId"].(string)
	if sourceOk {
		query = query.Where(sq.Eq{"locations.public_id": source.PublicID})
	} else if locationIDOk {
		query = query.Where(sq.Eq{"locations.public_id": locationID})
	}

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	receipts := []ReceiptWithData{}

	rows, err := r.db.Queryx(queryString, queryStringArgs...)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		receipt := ReceiptWithData{}
		var totalPrice sql.NullFloat64

		err := rows.Scan(&receipt.PublicID, &receipt.Location.PublicID, &receipt.CreatedBy, &receipt.Location.Name, &receipt.Location.Address, &receipt.CreatedAt, &receipt.UpdatedAt, &totalPrice)

		receipt.TotalPrice = totalPrice.Float64

		if err != nil {
			return nil, err
		}

		receipts = append(receipts, receipt)
	}

	return receipts, nil
}

// ItemsInReceiptResolver gets items from a specific receipt
func (r *Resolver) ItemsInReceiptResolver(p graphql.ResolveParams) (interface{}, error) {
	query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.created_by": r.userID})

	source, sourceOk := p.Source.(ReceiptWithData)
	receiptPublicID, receiptPublicIDOk := p.Args["receiptId"].(string)
	if sourceOk {
		query = query.Where(sq.Eq{"receipts.public_id": source.PublicID})
	} else if receiptPublicIDOk {
		query = query.Where(sq.Eq{"receipts.public_id": receiptPublicID})
	} else {
		return nil, fmt.Errorf("Could not parse receipt id!")
	}

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	items := []ItemInReceipt{}
	if err := r.db.Select(&items, queryString, queryStringArgs...); err != nil {
		return nil, err
	}

	return items, nil
}
