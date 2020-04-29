package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// GetLocationHandler is a Gin handler function for getting locations.
func GetLocationHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var searchQuery LocationsGetQuery
		if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		query := sq.Select("public_id, name, address, created_at, updated_at").From("locations").Where(sq.Eq{"created_by": user.ID})

		if searchQuery.Name != "" {
			query = query.Where("name LIKE ?", fmt.Sprint("%", searchQuery.Name, "%"))
		}

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		locations := []Location{}
		if err := db.Select(&locations, queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.JSON(http.StatusOK, locations)
	}
}

// PostLocationHandler is a Gin handler function for adding new locations.
func PostLocationHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var locationData LocationsPostBody
		if err := ctx.ShouldBindJSON(&locationData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		query := sq.Insert("locations").Columns("public_id", "name", "address", "created_by").Values(uuid, locationData.Name, locationData.Address, user.ID)

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

// PutLocationHandler is a Gin handler function for updating a location.
func PutLocationHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var locationData LocationsPutBody
		if err := ctx.ShouldBindJSON(&locationData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(locationData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		userOwnsQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": locationData.PublicID, "created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		var location StructID
		if err := db.Get(&location, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			switch err {
			case sql.ErrNoRows:
				ctx.String(http.StatusUnauthorized, "Not authrized to delete specified item from receipt.")
				break
			default:
				ctx.String(http.StatusInternalServerError, err.Error())
			}
			return
		}

		query := sq.Update("locations")

		if locationData.Name != "" {
			query = query.Set("name", locationData.Name)
		}
		if locationData.Address != "" {
			query = query.Set("address", locationData.Address)
		}

		query = query.Set("updated_at", time.Now())

		queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": locationData.PublicID}).ToSql()
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

// DeleteLocationHandler is a Gin handler function for deleting a location.
func DeleteLocationHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		createdBy, createdByExists := GetUserID(ctx)
		if !createdByExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		var locationData LocationsDeleteBody
		if err := ctx.ShouldBindJSON(&locationData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(locationData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user := PublicToPrivateUserID(db, createdBy)

		userOwnsQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": locationData.PublicID, "created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if err := db.Get(nil, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			ctx.String(http.StatusUnauthorized, "Not authrized to delete specified location.")
			return
		}

		query := sq.Delete("locations").Where(sq.Eq{"public_id": locationData.PublicID})
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
