package main

import (
	"fmt"
	"log"
	"context"
	"time"

	// Server related stuff
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
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
	router.Use(cors.Default())

	// db, err := sqlx.Connect("mysql", "root:rootpass@/receipts?parseTime=true")
	db, err := generateDatabase()
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

			queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": locationData.PublicID}).ToSql()
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

			query := sq.Delete("locations").Where(sq.Eq{"public_id": locationData.PublicID})
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

		// Get list of items from a specific receipt
		items.GET("/inreceipt/:id", func (ctx *gin.Context) {
			receiptPublicID := ctx.Param("id")
			if receiptPublicID == "" {
				ctx.String(http.StatusBadRequest, "Receipt id must be specified!")
				return
			}

			query := sq.Select("items_in_receipt.public_id, items.public_id as item_public_id, items.name as item_name, items.price as item_price, items.unit as item_unit, items_in_receipt.amount").From("items_in_receipt").Join("items ON items.id = items_in_receipt.item_id").Join("receipts ON receipts.id = items_in_receipt.receipt_id").Where(sq.Eq{"receipts.public_id": receiptPublicID})

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			items := []ItemInReceipt{}
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

			userIDQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": itemData.CreatedBy})
			userIDQueryString, userIDQueryStringArgs, err := userIDQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			user := UserID{}
			if err := db.Get(&user, userIDQueryString, userIDQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err.Error())
			}

			// fmt.Println(itemData)

			query := sq.Insert("items").Columns("public_id", "created_by", "name", "price", "unit").Values(uuid, user.ID, itemData.Name, itemData.Price, itemData.Unit)

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

		// Add item to receipts
		items.POST("/inreceipt", func (ctx *gin.Context) {
			var itemData ItemsPostToReceiptBody
			if err := ctx.ShouldBindJSON(&itemData); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			err := v.Struct(itemData)
			if err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}

			receiptIDQuery := sq.Select("id").From("receipts").Where(sq.Eq{"public_id": itemData.ReceiptID})

			receiptIDQueryString, receiptIDQueryStringArgs, err := receiptIDQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			receipt := ReceiptID{}
			if err := db.Get(&receipt, receiptIDQueryString, receiptIDQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			itemIDQuery := sq.Select("id").From("items").Where(sq.Eq{"public_id": itemData.ItemID})

			itemIDQueryString, itemIDQueryStringArgs, err := itemIDQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			item := ItemID{}
			if err := db.Get(&item, itemIDQueryString, itemIDQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err.Error())
			}

			query := sq.Insert("items_in_receipt").Columns("public_id", "receipt_id", "item_id", "amount").Values(uuid, receipt.ID, item.ID, itemData.Amount)

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

			query = query.Set("updated_at", time.Now())

			queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicID}).ToSql()
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

		// Update item from specific receipt
		items.PUT("/inreceipt", func (ctx *gin.Context) {
			var itemData ItemsInReceiptPutBody
			if err := ctx.ShouldBindJSON(&itemData); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
			}

			query := sq.Update("items_in_receipt")

			if itemData.Amount != 0.0 {
				query = query.Set("amount", itemData.Amount)
			}

			queryString, queryStringArgs, err := query.Where(sq.Eq{"public_id": itemData.PublicID}).ToSql()
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

			query := sq.Delete("items").Where(sq.Eq{"public_id": itemData.PublicID})
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

			if searchQuery.PublicID == "" && searchQuery.CreatedBy == "" && searchQuery.LocationID == "" {
				ctx.String(http.StatusBadRequest, "No search parameters specified!")
				return
			}

			if searchQuery.PublicID != "" && (searchQuery.CreatedBy != "" || searchQuery.LocationID != "") {
				ctx.String(http.StatusBadRequest, "Too many parameters specified!")
				return
			}

			query := sq.Select("receipts.public_id, locations.public_id AS location_id, users.public_id AS created_by, locations.name AS name, locations.address AS address, receipts.created_at, receipts.updated_at, SUM(items.price * items_in_receipt.amount) AS total_price").From("receipts").Join("locations ON locations.id = receipts.location_id").Join("users ON users.id = receipts.created_by").Join("items_in_receipt ON items_in_receipt.receipt_id = receipts.id").Join("items ON items.id = items_in_receipt.item_id")

			if searchQuery.PublicID != "" {
				query = query.Where(sq.Eq{"receipts.public_id": searchQuery.PublicID})
			} else {
				if searchQuery.CreatedBy != "" {
					query = query.Where(sq.Eq{"users.public_id": searchQuery.CreatedBy})
				}
				if searchQuery.LocationID != "" {
					query = query.Where(sq.Eq{"locations.public_id": searchQuery.LocationID})
				}
			}

			queryString, queryStringArgs, err := query.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			receipts := []ReceiptWithData{}
			rows, err := db.Queryx(queryString, queryStringArgs...)
			for rows.Next() {
				receipt := ReceiptWithData{}
				err := rows.Scan(&receipt.PublicID, &receipt.Location.PublicID, &receipt.CreatedBy, &receipt.Location.Name, &receipt.Location.Address, &receipt.CreatedAt, &receipt.UpdatedAt, &receipt.TotalPrice)
				if err != nil {
					log.Fatalln(err.Error())
				}
				receipts = append(receipts, receipt)
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

			locationIDQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": receiptData.LocationPublicID})
			locationIDQueryString, locationIDQueryStringArgs, err := locationIDQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			location := LocationID{}
			if err := db.Get(&location, locationIDQueryString, locationIDQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			userIDQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": receiptData.CreatedByPublicID})
			userIDQueryString, userIDQueryStringArgs, err := userIDQuery.ToSql()
			if err != nil {
				log.Fatalln(err.Error())
			}

			user := UserID{}
			if err := db.Get(&user, userIDQueryString, userIDQueryStringArgs...); err != nil {
				log.Fatalln(err.Error())
			}

			uuid, err := nanoid.Nanoid()
			if err != nil {
				log.Fatalln(err.Error())
			}

			query := sq.Insert("receipts").Columns("public_id", "location_id", "created_by").Values(uuid, location.ID, user.ID)

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

		// Update receipt
		receipts.PUT("", func (ctx *gin.Context) {
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

			query := sq.Update("receipts")

			if receiptData.LocationID != "" {
				locationQuery := sq.Select("id").From("locations").Where(sq.Eq{"public_id": receiptData.LocationID})

				locationQueryString, locationQueryStringArgs, err := locationQuery.ToSql()
				if err != nil {
					log.Fatalln(err.Error())
				}

				location := LocationID{}
				if err := db.Get(&location, locationQueryString, locationQueryStringArgs...); err != nil {
					log.Fatalln(err.Error())
				}

				query = query.Set("location_id", location.ID)
			}

			query = query.Set("updated_at", time.Now()).Where(sq.Eq{"public_id": receiptData.PublicID})

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

		// Delete receipt
		receipts.DELETE("", func (ctx *gin.Context) {
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

			query := sq.Delete("receipts").Where(sq.Eq{"public_id": receiptData.PublicID})

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

	router.Run(":3001")
}
