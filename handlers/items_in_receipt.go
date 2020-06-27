package handlers

import (
	"database/sql"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/jkomyno/nanoid"
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

// GetItemsInReceipt is a Gin handler function for getting items from
// a specific receipt.
func (o Options) GetItemsInReceipt() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		receiptPublicID := ctx.Param("id")
		if receiptPublicID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Receipt id must be specified!",
			})
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.public_id": receiptPublicID, "receipts.created_by": user.ID})

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		items := []ItemInReceipt{}
		if err := o.DB.Select(&items, queryString, queryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, items)
	}
}

// PostItemsInReceipt is a Gin handler function for adding new items to
// a specific receipt.
func (o Options) PostItemsInReceipt() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsInReceiptPostBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		err := o.V.Struct(itemData)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		receiptIDQuery := sq.Select("id").From("receipts").Where(sq.Eq{"public_id": itemData.ReceiptID, "created_by": user.ID})

		receiptIDQueryString, receiptIDQueryStringArgs, err := receiptIDQuery.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		receipt := StructID{}
		if err := o.DB.Get(&receipt, receiptIDQueryString, receiptIDQueryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		itemIDQuery := sq.Select("id").From("items").Where(sq.Eq{"public_id": itemData.ItemID, "created_by": user.ID})

		itemIDQueryString, itemIDQueryStringArgs, err := itemIDQuery.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}


		item := StructID{}
		if err := o.DB.Get(&item, itemIDQueryString, itemIDQueryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		query := sq.Insert("items_in_receipt").Columns("public_id", "receipt_id", "item_id", "amount").Values(uuid, receipt.ID, item.ID, itemData.Amount)

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		tx, err := o.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := tx.Commit(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		ctx.Status(http.StatusOK)
	}
}

// PutItemsInReceipt is a Gin handler function for updating items in a
// specific receipt.
func (o Options) PutItemsInReceipt() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsInReceiptPutBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		userOwnsQuery := sq.Select("items_in_receipt.id").From("items_in_receipt").Join("receipts on receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"items_in_receipt.public_id": itemData.PublicID, "receipts.created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		item := StructID{}
		if err := o.DB.Get(&item, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "not authrized to edit specified item from receipt",
			})
			return
		}

		query := sq.Update("items_in_receipt")

		if itemData.Amount != 0.0 {
			query = query.Set("amount", itemData.Amount)
		}

		queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicID}).ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		tx, err := o.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			log.Fatalln(err.Error())
		}

		tx.Commit()

		ctx.Status(http.StatusOK)
	}
}

// DeleteItemsInReceipt is a Gin handler function for deleting items from
// a specific receipt.
func (o Options) DeleteItemsInReceipt() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsInReceiptDeleteBody
		if err := ctx.ShouldBindJSON(&itemData); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		err := o.V.Struct(itemData)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		userOwnsQuery := sq.Select("items_in_receipt.id").From("items_in_receipt").Join("receipts ON receipts.id = items_in_receipt.receipt_id")

		if itemData.ReceiptID == "" {
			userOwnsQuery = userOwnsQuery.Where(sq.Eq{"items_in_receipt.public_id": itemData.ItemID, "receipts.created_by": user.ID})
		} else {
			userOwnsQuery = userOwnsQuery.Where(sq.Eq{"items_in_receipt.item_id": itemData.ItemID, "items_in_receipt.receipt_id": itemData.ReceiptID, "receipts.created_by": user.ID})
		}

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()

		var item StructID
		if err := o.DB.Get(&item, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			switch err {
			case sql.ErrNoRows:
				ctx.JSON(http.StatusUnauthorized, gin.H{
					"message": "not authrized to delete specified item from receipt",
				})
				break
			default:
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
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
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		tx, err := o.DB.Begin()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := tx.Commit(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		ctx.Status(http.StatusOK)
	}
}
