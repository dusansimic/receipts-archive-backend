package resolvers

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/dusansimic/receipts-archive-backend/handlers"
	graphql "github.com/graph-gophers/graphql-go"
)

// ReceiptWithDataAndItems is a struct for storing both receipt with its data
// and items from that receipt. This struct is used only in receipt resolver.
type ReceiptWithDataAndItems struct {
	handlers.ReceiptWithData
	items []handlers.ItemInReceipt
}

// ReceiptResolver is a struct for resolved receipt
type ReceiptResolver struct {
	receipt ReceiptWithDataAndItems
}

// ReceiptResolverArgs is a struct for receipt resolver arguments
type ReceiptResolverArgs struct {
	LocationID *string
}

func hasField(ctx context.Context, fieldname string) bool {
	for _, field := range graphql.SelectedFieldsFromContext(ctx) {
		if field.Name == fieldname {
			return true
		}
	}

	return false
}

// Receipts is a receipts resolver. If locationId argument is specified it
// gets receipts from a specified location
func (r *Resolver) Receipts(ctx context.Context, args ReceiptResolverArgs) (*[]*ReceiptResolver, error) {
	publicID := GetUserID(ctx)
	locationID := args.LocationID

	query := sq.Select("receipts.public_id, locations.public_id AS location_id, users.public_id AS created_by, locations.name AS name, locations.address AS address, receipts.created_at, receipts.updated_at, ROUND(SUM(items.price * items_in_receipt.amount), 2) AS total_price").From("receipts").Join("locations ON locations.id = receipts.location_id").Join("users ON users.id = receipts.created_by").LeftJoin("items_in_receipt ON items_in_receipt.receipt_id = receipts.id").LeftJoin("items ON items.id = items_in_receipt.item_id").GroupBy("receipts.id").Where(sq.Eq{"users.public_id": publicID.PublicID})

	if locationID != nil {
		query = query.Where(sq.Eq{"locations.public_id": locationID})
	}

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	resolver := []*ReceiptResolver{}

	rows, err := r.db.Queryx(queryString, queryStringArgs...)
	if err != nil {
		return nil, err
	}

	user := handlers.StructID{}

	hasItemsField := hasField(ctx, "itemsInReceipt")
	if hasItemsField {
		user, err = publicID.PrivateID(r.db)
		if err != nil {
			return nil, err
		}
	}

	for rows.Next() {
		receipt := ReceiptWithDataAndItems{}
		var totalPrice sql.NullFloat64

		err := rows.Scan(&receipt.PublicID, &receipt.Location.PublicID, &receipt.CreatedBy, &receipt.Location.Name, &receipt.Location.Address, &receipt.CreatedAt, &receipt.UpdatedAt, &totalPrice)

		receipt.TotalPrice = totalPrice.Float64

		if err != nil {
			return nil, err
		}

		if hasItemsField {
			query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.created_by": user.ID, "receipts.public_id": receipt.PublicID})

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				return nil, err
			}

			if err := r.db.Select(&receipt.items, queryString, queryStringArgs...); err != nil {
				return nil, err
			}
		}

		resolver = append(resolver, &ReceiptResolver{
			receipt: receipt,
		})
	}

	return &resolver, nil
}

// ID gets the id field from receipt
func (r *ReceiptResolver) ID() string {
	return r.receipt.PublicID
}

// CreatedBy get the createdBy field from receipt
func (r *ReceiptResolver) CreatedBy() string {
	return r.receipt.CreatedBy
}

// Location get the location field from receipt
func (r *ReceiptResolver) Location() *LocationResolver {
	return &LocationResolver{
		location: r.receipt.Location,
	}
}

// TotalPrice gets the totalPrice field from receipt
func (r *ReceiptResolver) TotalPrice() float64 {
	return r.receipt.TotalPrice
}

// CreatedAt gets the createdAt field from receipt
func (r *ReceiptResolver) CreatedAt() graphql.Time {
	return graphql.Time{
		Time: r.receipt.CreatedAt,
	}
}

// UpdatedAt gets the updatedAt field from receipt
func (r *ReceiptResolver) UpdatedAt() graphql.Time {
	return graphql.Time{
		Time: r.receipt.UpdatedAt,
	}
}

// ItemsInReceipt gets the items from a specific receipt for a receipt field
func (r *ReceiptResolver) ItemsInReceipt() *[]*ItemInReceiptResolver {
	items := []*ItemInReceiptResolver{}
	for _, item := range r.receipt.items {
		items = append(items, &ItemInReceiptResolver{
			itemInReceipt: item,
		})
	}

	return &items
}
