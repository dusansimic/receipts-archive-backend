package main

import (
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

var schema = `
scalar Time

type Query {
	locations(name: String): [Location!]
	receipts(locationId: String): [Receipt!]
	itemsInReceipt(receiptId: String): [ItemInReceipt!]
}

type Location {
	id: String!
	name: String!
	address: String!
	createdAt: Time!
	updatedAt: Time!
}

type Receipt {
	id: String!
	createdBy: String!
	location: Location!
	totalPrice: Float!
	createdAt: Time!
	updatedAt: Time!
	itemsInReceipt: [ItemInReceipt]
}

type ItemInReceipt {
	id: String!
	itemId: String!
	name: String!
	price: Float!
	unit: String!
	amount: Float!
}
`

// Resolver struct for storing required data
type Resolver struct {
	db *sqlx.DB
}

// NewSchema creates a new schema based on schema type and query struct
func NewSchema(db *sqlx.DB) *graphql.Schema {
	resolver := Resolver{
		db: db,
	}
	return graphql.MustParseSchema(schema, &resolver)
}

// GraphQLBody is a struct for storing graphql request data
type GraphQLBody struct {
	Query string `json:"query"`
	OperationName string `json:"operationName"`
	Variables string `json:"variables"`
}

// GraphQLHandler handles grpahql requests
func GraphQLHandler(db *sqlx.DB) gin.HandlerFunc {
	return gin.WrapH(&relay.Handler{Schema: NewSchema(db)})
}
