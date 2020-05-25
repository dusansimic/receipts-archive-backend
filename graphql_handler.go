package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/jmoiron/sqlx"
)

var locationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Location",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"address": &graphql.Field{
				Type: graphql.String,
			},
			"createdAt": &graphql.Field{
				Type: graphql.DateTime,
			},
			"updatedAt": &graphql.Field{
				Type: graphql.DateTime,
			},
		},
	},
)

// NewReceiptType creates a new receipt object for graphql
func NewReceiptType(resolver Resolver) *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Receipt",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.String,
				},
				"createdBy": &graphql.Field{
					Type: graphql.String,
				},
				"location": &graphql.Field{
					Type: locationType,
				},
				"totalPrice": &graphql.Field{
					Type: graphql.Float,
				},
				"createdAt": &graphql.Field{
					Type: graphql.DateTime,
				},
				"updatedAt": &graphql.Field{
					Type: graphql.DateTime,
				},
				"itemsInReceipt": &graphql.Field{
					Type: graphql.NewList(itemInReceiptType),
					Resolve: resolver.ItemsInReceiptResolver,
				},
			},
		},
	)
}

var itemInReceiptType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ItemInReceipt",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"itemId": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"price": &graphql.Field{
				Type: graphql.Float,
			},
			"unit": &graphql.Field{
				Type: graphql.String,
			},
			"amount": &graphql.Field{
				Type: graphql.Float,
			},
		},
	},
)

// NewQuery creates new root query
func NewQuery(userID int, db *sqlx.DB) *graphql.Object {
	resolver := Resolver{
		userID: userID,
		db: db,
	}

	queryType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"locations": &graphql.Field{
					Type: graphql.NewList(locationType),
					Description: "Get locations by name",
					Args: graphql.FieldConfigArgument{
						"name": &graphql.ArgumentConfig{
							Type: graphql.String,
							Description: "Part of the name of a location",
							DefaultValue: "",
						},
					},
					Resolve: resolver.LocationsResolver,
				},
				"receipts": &graphql.Field{
					Type: graphql.NewList(NewReceiptType(resolver)),
					Description: "Get receipts by location id",
					Args: graphql.FieldConfigArgument{
						"locationId": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
					Resolve: resolver.ReceiptsResolver,
				},
				"itemsInReceipt": &graphql.Field{
					Type: graphql.NewList(itemInReceiptType),
					Description: "Get items from a specific receipt",
					Args: graphql.FieldConfigArgument{
						"receiptId": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
					Resolve: resolver.ItemsInReceiptResolver,
				},
			},
		},
	)

	return queryType
}

// NewSchema creates a new schema based on query object
func NewSchema(userID int, db *sqlx.DB) (graphql.Schema, error) {
	return graphql.NewSchema(
		graphql.SchemaConfig{
			Query: NewQuery(userID, db),
		},
	)
}

// GraphQLBody is a struct for storing graphql request data
type GraphQLBody struct {
	Query string `json:"query"`
	OperationName string `json:"operationName"`
	Variables string `json:"variables"`
}

// GraphQLHandler handles grpahql requests
func GraphQLHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		userID, userIDExists := GetUserID(ctx)
		if !userIDExists {
			ctx.String(http.StatusUnauthorized, "User id not found in authorization token.")
			return
		}

		user := PublicToPrivateUserID(db, userID)

		schema, err := NewSchema(user.ID, db)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		var body GraphQLBody
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		result := graphql.Do(graphql.Params{
			Schema: schema,
			RequestString: body.Query,
		})

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Println(err.Error())
			}
			ctx.Status(http.StatusInternalServerError)
			return
		}

		ctx.JSON(http.StatusOK, result)
	}
}
