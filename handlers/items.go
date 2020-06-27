package handlers

import (
	"fmt"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/jkomyno/nanoid"
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

// GetItems is a Gin handler function for getting items.
func (o Options) GetItems() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var searchQuery ItemsGetQuery
		if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		query := sq.Select("items.public_id, name, price, unit, created_at, updated_at").From("items")

		query = query.Join("users ON users.id = items.created_by").Where(sq.Eq{"users.public_id": createdBy.PublicID})

		if searchQuery.Name != "" {
			query = query.Where("items.name LIKE ?", fmt.Sprint("%", searchQuery.Name, "%"))
		}

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		items := []Item{}
		if err := o.DB.Select(&items, queryString, queryStringArgs...); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		}

		ctx.JSON(http.StatusOK, items)
	}
}

// PostItems is a Gin handler function for adding new items.
func (o Options) PostItems() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsPostBody
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

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		query := sq.Insert("items").Columns("public_id", "created_by", "name", "price", "unit").Values(uuid, user.ID, itemData.Name, itemData.Price, itemData.Unit)

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

// PutItems is a Gin handler function for updating items.
func (o Options) PutItems() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsPutBody
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

// DeleteItems is a Gin handler function for deleting items.
func (o Options) DeleteItems() gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "user id not found in authorization token",
			})
			return
		}

		var itemData ItemsDeleteBody
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

		query := sq.Delete("items").Where(sq.Eq{"public_id": itemData.PublicID, "created_by": user.ID})
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
		}

		tx.Commit()

		ctx.Status(http.StatusOK)
	}
}
