package main

import (
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
)

// ReceiptsGetQuery : Structure that should be used for getting query data on get request for receipts
type ReceiptsGetQuery struct {
	PublicID string `form:"id"`
	LocationID string `form:"locationId"`
}

// ReceiptsPostBody : Structure that should be used for getting json from body of a post request for receipts
type ReceiptsPostBody struct {
	LocationPublicID string `json:"id" validate:"required"`
	CreatedAt string `json:"createdAt"`
}

// ReceiptsPutBody : Structure that should be used for getting json from body of a put request for receipts
type ReceiptsPutBody struct {
	PublicID string `json:"id" validate:"required"`
	LocationID string `json:"locationId"`
}

// ReceiptsDeleteBody : Structure that should be used for getting json data from body of a delete request for items
type ReceiptsDeleteBody struct {
	PublicID string `json:"id" validate:"required"`
}

// Receipt : Structure that should be used for getting receipt information from database
type Receipt struct {
	PublicID string `db:"public_id" json:"id"`
	LocationID string `db:"location_id" json:"locationId"`
	CreatedBy string `db:"created_by" json:"createdBy"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// ReceiptWithData : Structure that should be used for getting receipt information including names, addresses, and everything else from receipts location from database
type ReceiptWithData struct {
	PublicID string `json:"id"`
	CreatedBy string `json:"createdBy"`
	Location Location `json:"location"`
	TotalPrice float64 `json:"totalPrice"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func GetReceiptsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var searchQuery ReceiptsGetQuery
		if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		if searchQuery.PublicID == "" && createdBy == "" && searchQuery.LocationID == "" {
			ctx.String(http.StatusBadRequest, "No search parameters specified!")
			return
		}

		if searchQuery.PublicID != "" && searchQuery.LocationID != "" {
			ctx.String(http.StatusBadRequest, "Too many parameters specified!")
			return
		}

		query := sq.Select("receipts.public_id, locations.public_id AS location_id, users.public_id AS created_by, locations.name AS name, locations.address AS address, receipts.created_at, receipts.updated_at, SUM(items.price * items_in_receipt.amount) AS total_price").From("receipts").Join("locations ON locations.id = receipts.location_id").Join("users ON users.id = receipts.created_by").LeftJoin("items_in_receipt ON items_in_receipt.receipt_id = receipts.id").LeftJoin("items ON items.id = items_in_receipt.item_id").GroupBy("receipts.id")

		if searchQuery.PublicID != "" {
			query = query.Where(sq.Eq{"receipts.public_id": searchQuery.PublicID})
		} else {
			if createdBy != "" {
				query = query.Where(sq.Eq{"users.public_id": createdBy})
			}
			if searchQuery.LocationID != "" {
				query = query.Where(sq.Eq{"locations.public_id": searchQuery.LocationID})
			}
		}

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		receipts := []ReceiptWithData{}
		rows, err := db.Queryx(queryString, queryStringArgs...)

		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		for rows.Next() {
			receipt := ReceiptWithData{}
			err := rows.Scan(&receipt.PublicID, &receipt.Location.PublicID, &receipt.CreatedBy, &receipt.Location.Name, &receipt.Location.Address, &receipt.CreatedAt, &receipt.UpdatedAt, &receipt.TotalPrice)

			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}

			receipts = append(receipts, receipt)
		}

		ctx.JSON(http.StatusOK, receipts)
	}
}

func PostReceiptsHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var receiptData ReceiptsPostBody
		if err := ctx.ShouldBindJSON(&receiptData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(receiptData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		locationIDQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": receiptData.LocationPublicID})
		locationIDQueryString, locationIDQueryStringArgs, err := locationIDQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		location := StructID{}
		if err := db.Get(&location, locationIDQueryString, locationIDQueryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		createdAt, updatedAt := time.Now(), time.Now()
		if receiptData.CreatedAt != "" {
			createdAt, err = time.Parse(time.RFC3339, receiptData.CreatedAt)
			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}

			updatedAt, err = time.Parse(time.RFC3339, receiptData.CreatedAt)
			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}
		}

		query := sq.Insert("receipts").Columns("public_id", "location_id", "created_by", "created_at", "updated_at").Values(uuid, location.ID, user.ID, createdAt, updatedAt)

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

func PutReceiptsHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var receiptData ReceiptsPutBody
		if err := ctx.ShouldBindJSON(&receiptData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(receiptData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		query := sq.Update("receipts")

		if receiptData.LocationID != "" {
			locationQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": receiptData.LocationID})

			locationQueryString, locationQueryStringArgs, err := locationQuery.ToSql()
			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}

			location := StructID{}
			if err := db.Get(&location, locationQueryString, locationQueryStringArgs...); err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}

			query = query.Set("location_id", location.ID)
		}

		query = query.Set("updated_at", time.Now()).Where(sq.Eq{"public_id": receiptData.PublicID, "created_by": user.ID})

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

func DeleteReceiptsHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var receiptData ReceiptsDeleteBody
		if err := ctx.ShouldBindJSON(&receiptData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(receiptData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		query := sq.Delete("receipts").Where(sq.Eq{"public_id": receiptData.PublicID, "created_by": user.ID})

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
