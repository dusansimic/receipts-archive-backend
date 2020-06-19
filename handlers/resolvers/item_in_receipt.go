package resolvers

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/dusansimic/receipts-archive-backend/handlers"
)

// ItemInReceiptResolver is a struct for resolver itemInReceipt
type ItemInReceiptResolver struct {
	itemInReceipt handlers.ItemInReceipt
}

// ItemInReceiptResolverArgs is a struct for itemInReceipt resolver arguments
type ItemInReceiptResolverArgs struct {
	ReceiptID *string
}

// ItemsInReceipt is a itemInReceipt resolver. If receiptId arguemnt is
// specified, it gets items from a specified receipt, otherwise it throws an
// error.
func (r *Resolver) ItemsInReceipt(ctx context.Context, args ItemInReceiptResolverArgs) (*[]*ItemInReceiptResolver, error) {
	receiptID := args.ReceiptID

	publicID := GetUserID(ctx)
	user, err := publicID.PrivateID(r.db)
	if err != nil {
		return nil, err
	}

	query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.created_by": user.ID})

	if receiptID != nil {
		query = query.Where(sq.Eq{"receipts.public_id": receiptID})
	}

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	resolver := []*ItemInReceiptResolver{}

	items := []handlers.ItemInReceipt{}
	if err := r.db.Select(&items, queryString, queryStringArgs...); err != nil {
		return nil, err
	}

	for _, item := range items {
		resolver = append(resolver, &ItemInReceiptResolver{
			itemInReceipt: item,
		})
	}

	return &resolver, nil
}

// ID gets the id field from itemInReceipt
func (r *ItemInReceiptResolver) ID() string {
	return r.itemInReceipt.PublicID
}

// ItemID gets the itemId field from itemInReceipt
func (r *ItemInReceiptResolver) ItemID() string {
	return r.itemInReceipt.ItemPublicID
}

// Name gets the name field from itemInReceipt
func (r *ItemInReceiptResolver) Name() string {
	return r.itemInReceipt.Name
}

// Price gets the price field from itemInReceipt
func (r *ItemInReceiptResolver) Price() float64 {
	return float64(r.itemInReceipt.Price)
}

// Unit gets the unit field from itemInReceipt
func (r *ItemInReceiptResolver) Unit() string {
	return r.itemInReceipt.Unit
}

// Amount gets the amount field from itemInReceipt
func (r *ItemInReceiptResolver) Amount() float64 {
	return float64(r.itemInReceipt.Amount)
}
