package main

import "time"

// UserID : Structure that should be used for getting just the ID from database
type UserID struct {
	ID int `db:"id"`
}

// User : Structure that should be used for getting user information from database
type User struct {
	PublicID string `db:"public_id"`
	RealName string `db:"read_name"`
}

// LocationsGetQuery : Structure that should be used for getting query data on get request for locations
type LocationsGetQuery struct {
	Name string `form:"name"`
}

// LocationsPostBody : Structure that should be used for getting json from body of a post request for locations
type LocationsPostBody struct {
	Name string `json:"name" validate:"required"`
	Address string `json:"address" validate:"required"`
	CreatedBy string `json:"userId" validate:"required"`
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

// LocationID : Structure that should be used for getting just the ID from database
type LocationID struct {
	ID int `db:"id"`
}

// Location : Structure that should be used for getting location information from database
type Location struct {
	PublicID string `db:"public_id" json:"id"`
	Name string `db:"name" json:"name"`
	Address string `db:"address" json:"address"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// ItemsGetQuery : Structure that should be used for getting query data on get request for items
type ItemsGetQuery struct {
	CreatedBy string `form:"createdBy"`
	Name string `form:"name"`
}

// ItemsPostBody : Structure that should be used for getting json from body of a post request for items
type ItemsPostBody struct {
	CreatedBy string `json:"createdBy" validate:"required"`
	Name string `json:"name" validate:"required"`
	Price float32 `json:"price" validate:"required"`
	Unit string `json:"unit" validate:"required"`
}

// ItemsPostToReceiptBody : Structure that should be used for getting json from body of a post request for adding item to a receipt
type ItemsPostToReceiptBody struct {
	ReceiptID string `json:"receiptId" validate:"required"`
	ItemID string `json:"itemId" validate:"required"`
	Amount float32 `json:"amount" validate:"required"`
}

// ItemsPutBody : Structure that should be used for getting json from body of a put request for items
type ItemsPutBody struct {
	PublicID string `json:"id" validate:"required"`
	Name string `json:"name"`
	Price float32 `json:"price"`
	Unit string `json:"unit"`
}

// ItemsInReceiptPutBody : Structure that should be used for getting json from body of a put request for items form a specific receipt
type ItemsInReceiptPutBody struct {
	PublicID string `json:"id" validate:"required"`
	Amount float32 `json:"amount"`
}

// ItemsDeleteBody : Structure that should be used for getting json data from body of a delete request for items
type ItemsDeleteBody struct {
	PublicID string `json:"id" validate:"required"`
}

// ItemID : Structure that should be used for getting just the ID from database
type ItemID struct {
	ID int `db:"id"`
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

// ReceiptsGetQuery : Structure that should be used for getting query data on get request for receipts
type ReceiptsGetQuery struct {
	PublicID string `form:"id"`
	CreatedBy string `form:"createdBy"`
	LocationID string `form:"locationId"`
}

// ReceiptsPostBody : Structure that should be used for getting json from body of a post request for receipts
type ReceiptsPostBody struct {
	LocationPublicID string `json:"locationId" validate:"required"`
	CreatedByPublicID string `json:"createdBy" validate:"required"`
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

// ReceiptID : Structure that should be used for getting just the ID from database
type ReceiptID struct {
	ID int `db:"id"`
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
	TotalPrice float32 `json:"totalPrice"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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