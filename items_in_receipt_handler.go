package main

import (
	"database/sql"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
)

// ItemsInReceiptPostBody : Structure that should be used for getting json from body of a post request for adding item to a receipt
type ItemsInReceiptPostBody struct {
	ReceiptID string `json:"receiptId" validate:"required"`
	ItemID string `json:"itemId" validate:"required"`
	Amount float32 `json:"amount" validate:"required"`
}

// ItemsInReceiptPutBody : Structure that should be used for getting json from body of a put request for items form a specific receipt
type ItemsInReceiptPutBody struct {
	PublicID string `json:"id" validate:"required"`
	Amount float32 `json:"amount"`
}

// ItemsInReceiptDeleteBody : Structure that should be used for getting json data from body of a delete request for items in a specific receipt
type ItemsInReceiptDeleteBody struct {
	ItemID string `json:"itemId" validate:"required"`
	ReceiptID string `json:"receiptId"`
}

// ItemInReceipt : Structure that should be used for getting item information of a specific receipt from database
type ItemInReceipt struct {
	PublicID string `db:"public_id" json:"id"`
	ItemPublicID string `db:"item_public_id" json:"itemId"`
	Name string `db:"item_name" json:"name"`
	Price float32 `db:"item_price" json:"price"`
	Unit string `db:"item_unit" json:"unit"`
	Amount float32 `db:"amount" json:"amount"`
}

// GetItemsInReceiptHandler is a Gin handler function for getting items from
// a specific receipt.
func GetItemsInReceiptHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		receiptPublicID := ctx.Param("id")
		if receiptPublicID == "" {
			ctx.String(http.StatusBadRequest, "Receipt id must be specified!")
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.public_id": receiptPublicID, "receipts.created_by": user.ID})

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		items := []ItemInReceipt{}
		if err := db.Select(&items, queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.JSON(http.StatusOK, items)
	}
}

// PostItemsInReceiptHandler is a Gin handler function for adding new items to
// a specific receipt.
func PostItemsInReceiptHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsInReceiptPostBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(itemData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		receiptIDQuery := sq.Select("id").From("receipts").Where(sq.Eq{"public_id": itemData.ReceiptID, "created_by": user.ID})

		receiptIDQueryString, receiptIDQueryStringArgs, err := receiptIDQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		receipt := StructID{}
		if err := db.Get(&receipt, receiptIDQueryString, receiptIDQueryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		itemIDQuery := sq.Select("id").From("items").Where(sq.Eq{"public_id": itemData.ItemID, "created_by": user.ID})

		itemIDQueryString, itemIDQueryStringArgs, err := itemIDQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		item := StructID{}
		if err := db.Get(&item, itemIDQueryString, itemIDQueryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		query := sq.Insert("items_in_receipt").Columns("public_id", "receipt_id", "item_id", "amount").Values(uuid, receipt.ID, item.ID, itemData.Amount)

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		tx, err := db.Begin()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if err := tx.Commit(); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Status(http.StatusOK)
	}
}

// PutItemsInReceiptHandler is a Gin handler function for updating items in a
// specific receipt.
func PutItemsInReceiptHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsInReceiptPutBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		userOwnsQuery := sq.Select("items_in_receipt.id").From("items_in_receipt").Join("receipts on receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"items_in_receipt.public_id": itemData.PublicID, "receipts.created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		item := StructID{}
		if err := db.Get(&item, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			ctx.String(http.StatusUnauthorized, "Not authrized to edit specified item from receipt.")
			return
		}

		query := sq.Update("items_in_receipt")

		if itemData.Amount != 0.0 {
			query = query.Set("amount", itemData.Amount)
		}

		queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicID}).ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		tx, err := db.Begin()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			log.Fatalln(err.Error())
		}

		tx.Commit()

		ctx.Status(http.StatusOK)
	}
}

// DeleteItemsInReceiptHandler is a Gin handler function for deleting items from
// a specific receipt.
func DeleteItemsInReceiptHandler (db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsInReceiptDeleteBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(itemData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		userOwnsQuery := sq.Select("items_in_receipt.id").From("items_in_receipt").Join("receipts ON receipts.id = items_in_receipt.receipt_id")

		if itemData.ReceiptID == "" {
			userOwnsQuery = userOwnsQuery.Where(sq.Eq{"items_in_receipt.public_id": itemData.ItemID, "receipts.created_by": user.ID})
		} else {
			userOwnsQuery = userOwnsQuery.Where(sq.Eq{"items_in_receipt.item_id": itemData.ItemID, "items_in_receipt.receipt_id": itemData.ReceiptID, "receipts.created_by": user.ID})
		}

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()

		var item StructID
		if err := db.Get(&item, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			switch err {
			case sql.ErrNoRows:
				ctx.String(http.StatusUnauthorized, "Not authrized to delete specified item from receipt.")
				break
			default:
				ctx.String(http.StatusInternalServerError, err.Error())
			}
			return
		}

		query := sq.Delete("items_in_receipt")

		if itemData.ReceiptID == "" {
			query = query.Where(sq.Eq{"public_id": itemData.ItemID})
		} else {
			query = query.Where(sq.Eq{"item_id": itemData.ItemID, "receipt_id": itemData.ReceiptID})
		}

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		tx, err := db.Begin()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if err := tx.Commit(); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Status(http.StatusOK)
	}
}
