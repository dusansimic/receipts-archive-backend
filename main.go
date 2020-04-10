package main

import (
	"fmt"
	"log"
	"context"

	// Server related stuff
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions/cookie"

	// Auth stuff
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	// DB stuff
	sq "github.com/Masterminds/squirrel"
	// "database/sql"
	_ "github.com/mattn/go-sqlite3"
	// _ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	// Other stuff
	"github.com/jkomyno/nanoid"
	"github.com/go-playground/validator"
)

/*
 * Checks if there is a specified user in the database. If there is, does
 * nothing. If there is not, inserts the data in database.
 */
func userCheck(user goth.User, db *sqlx.DB) {
	query := sq.Select("public_id").From("users").Where(sq.Eq{"public_id": user.UserID})
	queryString, queryStringArgs, _ := query.ToSql()

	users := []User{}
	if err := db.Select(&users, queryString, queryStringArgs...); err != nil {
		log.Fatalln(err.Error())
	}

	if len(users) == 0 {
		insertQuery := sq.Insert("users").Columns("public_id", "real_name").Values(user.UserID, user.Email)
		insertQueryString, insertArgs, _ := insertQuery.ToSql()

		tx, err := db.Begin()
		if err != nil {
			log.Fatalln(err.Error())
		}

		if _, err := tx.Exec(insertQueryString, insertArgs...); err != nil {
			log.Fatalln(err.Error())
		}

		if err := tx.Commit(); err != nil {
			log.Fatalln(err.Error())
		}
	}
}

func main() {
	router := gin.Default()

	// db, err := sqlx.Connect("mysql", "root:rootpass@/receipts?parseTime=true")
	db, err := sqlx.Connect("sqlite3", "./receipts.db")
	if err != nil {
		fmt.Println("Failed to connect to the database!")
		log.Fatalln(err.Error())
		return
	}

	gothic.Store = cookie.NewStore([]byte("verysecretyes"))
	goth.UseProviders(google.New("944614249243-r9c42sh3ktt8vhrkf5t5maav6r4qr0a5.apps.googleusercontent.com", "LgyS_VCvznGW09cokooicK7v", "http://localhost:3000/auth/google/callback"))

	v := validator.New()

	auth := router.Group("/auth")
	{
		auth.GET("/google", func (ctx *gin.Context) {
			tmpContext := context.WithValue(ctx.Request.Context(), "provider", "google")
			newRequestContext := ctx.Request.WithContext(tmpContext)
			user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
			if err != nil {
				gothic.BeginAuthHandler(ctx.Writer, newRequestContext)
				return
			}

			userCheck(user, db)

			ctx.JSON(http.StatusOK, user)
		})

		auth.GET("/google/callback", func (ctx *gin.Context) {
			tmpContext := context.WithValue(ctx.Request.Context(), "provider", "google")
			newRequestContext := ctx.Request.WithContext(tmpContext)
			user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}

			userCheck(user, db)

			ctx.JSON(http.StatusOK, user)
		})
	}

	locations := router.Group("/locations")
	{
		// Get list of locations (query available)
		locations.GET("", func (ctx *gin.Context) {
			var searchQuery LocationsGetQuery
			if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			query := sq.Select("public_id, name, address, created_at, updated_at").From("locations")

			if searchQuery.Name != "" {
				query = query.Where("name LIKE ?", fmt.Sprint("%", searchQuery.Name, "%"))
			}

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			locations := []Location{}
			if err := db.Select(&locations, queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			ctx.JSON(http.StatusOK, locations)
		})

		// Add new location
		locations.POST("", func (ctx *gin.Context) {
			var locationData LocationsPostBody
			if err := ctx.ShouldBindJSON(&locationData); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err)
			}

			query := sq.Insert("locations").Columns("public_id", "name", "address", "created_by").Values(uuid, locationData.Name, locationData.Address, locationData.CreatedBy)

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})

		// Update location
		locations.PUT("", func (ctx *gin.Context) {
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

			query := sq.Update("locations")

			if locationData.Name != "" {
				query = query.Set("name", locationData.Name)
			}
			if locationData.Address != "" {
				query = query.Set("address", locationData.Address)
			}

			queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": locationData.PublicId}).ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})

		// Delete location
		locations.DELETE("", func (ctx *gin.Context) {
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

			query := sq.Delete("locations").Where(sq.Eq{"public_id": locationData.PublicId})
			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})
	}

	items := router.Group("/items")
	{
		// Get list of items (query available)
		items.GET("", func (ctx *gin.Context) {
			var searchQuery ItemsGetQuery
			if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			query := sq.Select("items.public_id, name, price, unit, created_at, updated_at").From("items")

			if searchQuery.CreatedBy != "" {
				query = query.Join("users ON users.id = items.created_by").Where(sq.Eq{"users.public_id": searchQuery.CreatedBy})
			}

			if searchQuery.Name != "" {
				query = query.Where("items.name LIKE ?", fmt.Sprint("%", searchQuery.Name, "%"))
			}

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			items := []Item{}
			if err := db.Select(&items, queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			ctx.JSON(http.StatusOK, items)
		})

		// Add new item
		items.POST("", func (ctx *gin.Context) {
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

			userIdQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": itemData.CreatedBy})
			userIdQueryString, userIdQueryStringArgs, err := userIdQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			user := UserId{}
			if err := db.Get(&user, userIdQueryString, userIdQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err.Error())
			}

			// fmt.Println(itemData)

			query := sq.Insert("items").Columns("public_id", "created_by", "name", "price", "unit").Values(uuid, user.Id, itemData.Name, itemData.Price, itemData.Unit)

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})

		// Update item
		items.PUT("", func (ctx *gin.Context) {
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

			queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicId}).ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})

		// Delete item
		items.DELETE("", func (ctx *gin.Context) {
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

			query := sq.Delete("items").Where(sq.Eq{"public_id": itemData.PublicId})
			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})
	}

	receipts := router.Group("/receipts")
	{
		// Get list of receipts (query available)
		receipts.GET("", func (ctx *gin.Context) {
			var searchQuery ReceiptsGetQuery
			if err := ctx.ShouldBindQuery(&searchQuery); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			if searchQuery.PublicId == "" && searchQuery.CreatedBy == "" && searchQuery.LocationId == "" {
				ctx.String(http.StatusBadRequest, "No search parameters specified!")
				return
			}

			if searchQuery.PublicId != "" && (searchQuery.CreatedBy != "" || searchQuery.LocationId != "") {
				ctx.String(http.StatusBadRequest, "Too many parameters specified!")
				return
			}

			if searchQuery.PublicId != "" {
				query := sq.Select("id").From("receipts").Where(sq.Eq{"receipts.public_id": searchQuery.PublicId})

				queryString, queryStringArgs, err := query.ToSql()
				if err != nil {
					log.Fatalln(err.Error())
				}

				receipt := ReceiptId{}
				if err := db.Get(&receipt, queryString, queryStringArgs...); err != nil {
					log.Fatalln(err.Error())
				}
				// .From("items_in_receipt").Join("receipts ON receipts.id = items_in_receipt.receipt_id")
				itemsQuery := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Where(sq.Eq{"items_in_receipt.receipt_id": receipt.Id})

				itemsQueryString, itemsQueryStringArgs, err := itemsQuery.ToSql()
				if err != nil {
					log.Fatalln(err.Error())
				}

				items := []Item{}
				if err := db.Select(&items, itemsQueryString, itemsQueryStringArgs...); err != nil {
					log.Fatalln(err.Error())
				}

				ctx.JSON(http.StatusOK, items)
				return
			}

			query := sq.Select("receipts.public_id, locations.public_id as location_id, users.public_id as created_by, receipts.created_at, receipts.updated_at").From("receipts").Join("locations ON locations.id = receipts.location_id").Join("users ON users.id = receipts.created_by")

			if searchQuery.CreatedBy != "" {
				query = query.Where(sq.Eq{"users.public_id": searchQuery.CreatedBy})
			}
			if searchQuery.LocationId != "" {
				query = query.Where(sq.Eq{"locations.public_id": searchQuery.LocationId})
			}

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}
			
			receipts := []Receipt{}
			if err := db.Select(&receipts, queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			ctx.JSON(http.StatusOK, receipts)
		})

		// Add new receipt
		receipts.POST("", func (ctx *gin.Context) {
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

			locationIdQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": receiptData.LocationPublicId})
			locationIdQueryString, locationIdQueryStringArgs, err := locationIdQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			location := LocationId{}
			if err := db.Get(&location, locationIdQueryString, locationIdQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			userIdQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": receiptData.CreatedByPublicId})
			userIdQueryString, userIdQueryStringArgs, err := userIdQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			user := UserId{}
			if err := db.Get(&user, userIdQueryString, userIdQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err.Error())
			}

			query := sq.Insert("receipts").Columns("public_id", "location_id", "created_by").Values(uuid, location.Id, user.Id)

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			tx, err := db.Begin()
			if err != nil {
				log.Fatalln(err.Error())
			}

			if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			tx.Commit()

			ctx.Status(http.StatusOK)
		})
	}

	router.Run(":3000")
}
