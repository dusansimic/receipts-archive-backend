package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/jkomyno/nanoid"

	// Import SQLite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// LocationsGetQuery : Structure that should be used for getting query data on get request for locations
type LocationsGetQuery struct {
	Name string `form:"name"`
}

// LocationsPostBody : Structure that should be used for getting json from body of a post request for locations
type LocationsPostBody struct {
	Name string `json:"name" validate:"required"`
	Address string `json:"address" validate:"required"`
}

// LocationsPutBody : Structure that should be used for getting json from body of a put request for locations
type LocationsPutBody struct {
	PublicID string `json:"id" validate:"required"`
	Name string `json:"name"`
	Address string `json:"address"`
}

// LocationsDeleteBody : Structure that should be used for getting json data from body of a delete request for locations
type LocationsDeleteBody struct {
	PublicID string `json:"id" validate:"required"`
}

// Location : Structure that should be used for getting location information from database
type Location struct {
	PublicID string `db:"public_id" json:"id"`
	Name string `db:"name" json:"name"`
	Address string `db:"address" json:"address"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// GetLocations is a Gin handler function for getting locations.
func (o Options) GetLocations() gin.HandlerFunc {
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

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

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
		if err := o.DB.Select(&locations, queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.JSON(http.StatusOK, locations)
	}
}

// PostLocations is a Gin handler function for adding new locations.
func (o Options) PostLocations() gin.HandlerFunc {
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

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

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

		tx, err := o.DB.Begin()
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

// PutLocations is a Gin handler function for updating a location.
func (o Options) PutLocations() gin.HandlerFunc {
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

		err := o.V.Struct(locationData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		userOwnsQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": locationData.PublicID, "created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		var location StructID
		if err := o.DB.Get(&location, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
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

		tx, err := o.DB.Begin()
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

// DeleteLocations is a Gin handler function for deleting a location.
func (o Options) DeleteLocations() gin.HandlerFunc {
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

		err := o.V.Struct(locationData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		user, err := createdBy.PrivateID(o.DB)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		userOwnsQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": locationData.PublicID, "created_by": user.ID})

		userOwnsQueryString, userOwnsQueryStringArgs, err := userOwnsQuery.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		var location StructID
		if err := o.DB.Get(&location, userOwnsQueryString, userOwnsQueryStringArgs...); err != nil {
			ctx.String(http.StatusUnauthorized, "Not authrized to delete specified location.")
			return
		}

		query := sq.Delete("locations").Where(sq.Eq{"public_id": locationData.PublicID})
		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		tx, err := o.DB.Begin()
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
