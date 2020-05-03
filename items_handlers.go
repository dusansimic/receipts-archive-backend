package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
)

// ItemsGetQuery : Structure that should be used for getting query data on get request for items
type ItemsGetQuery struct {
	// CreatedBy string `form:"createdBy"`
	Name string `form:"name"`
}

// ItemsPostBody : Structure that should be used for getting json from body of a post request for items
type ItemsPostBody struct {
	// CreatedBy string `json:"createdBy" validate:"required"`
	Name string `json:"name" validate:"required"`
	Price float32 `json:"price" validate:"required"`
	Unit string `json:"unit" validate:"required"`
}

// ItemsPutBody : Structure that should be used for getting json from body of a put request for items
type ItemsPutBody struct {
	PublicID string `json:"id" validate:"required"`
	Name string `json:"name"`
	Price float32 `json:"price"`
	Unit string `json:"unit"`
}

// ItemsDeleteBody : Structure that should be used for getting json data from body of a delete request for items
type ItemsDeleteBody struct {
	PublicID string `json:"id" validate:"required"`
}

// Item : Structure that should be used for getting item information from database
type Item struct {
	PublicID string `db:"public_id" json:"id"`
	Name string `db:"name" json:"name"`
	Price float32 `db:"price" json:"price"`
	Unit string `db:"unit" json:"unit"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// GetItemsHandler is a Gin handler function for getting items.
func GetItemsHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var searchQuery ItemsGetQuery
		if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		query := sq.Select("items.public_id, name, price, unit, created_at, updated_at").From("items")

		if createdBy != "" {
			query = query.Join("users ON users.id = items.created_by").Where(sq.Eq{"users.public_id": createdBy})
		}

		if searchQuery.Name != "" {
			query = query.Where("items.name LIKE ?", fmt.Sprint("%", searchQuery.Name, "%"))
		}

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		items := []Item{}
		if err := db.Select(&items, queryString, queryStringArgs...); err != nil {
			log.Fatalln(err.Error())
		}

		ctx.JSON(http.StatusOK, items)
	}
}

// PostItemsHandler is a Gin handler function for adding new items.
func PostItemsHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsPostBody
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

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		query := sq.Insert("items").Columns("public_id", "created_by", "name", "price", "unit").Values(uuid, user.ID, itemData.Name, itemData.Price, itemData.Unit)

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

// PutItemsHandler is a Gin handler function for updating items.
func PutItemsHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsPutBody
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

		query := sq.Update("items")

		if itemData.Name != "" {
			query = query.Set("name", itemData.Name)
		}
		if itemData.Price != 0.0 {
			query = query.Set("price", itemData.Price)
		}
		if itemData.Unit != "" {
			query = query.Set("unit", itemData.Unit)
		}

		query = query.Set("updated_at", time.Now())

		queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicID, "created_by": user.ID}).ToSql()
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

// DeleteItemsHandler is a Gin handler function for deleting items.
func DeleteItemsHandler (db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var itemData ItemsDeleteBody
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

		query := sq.Delete("items").Where(sq.Eq{"public_id": itemData.PublicID, "created_by": user.ID})
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
			log.Fatalln(err.Error())
		}

		tx.Commit()

		ctx.Status(http.StatusOK)
	}
}
