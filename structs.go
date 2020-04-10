package main

import "time"

type UserId struct {
	Id int `db:"id"`
}

type User struct {
	PublicId string `db:"public_id"`
	RealName string `db:"read_name"`
}

type LocationsGetQuery struct {
	Name string `form:"name"`
}

type LocationsPostBody struct {
	Name string `json:"name" validate:"required"`
	Address string `json:"address" validate:"required"`
	CreatedBy string `json:"userId" validate:"required"`
}

type LocationsPutBody struct {
	PublicId string `json:"id" validate:"required"`
	Name string `json:"name"`
	Address string `json:"address"`
}

type LocationsDeleteBody struct {
	PublicId string `json:"id" validate:"required"`
}

type LocationId struct {
	Id int `db:"id"`
}

type Location struct {
	PublicId string `db:"public_id"`
	Name string `db:"name"`
	Address string `db:"address"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ItemsGetQuery struct {
	CreatedBy string `form:"createdBy"`
	Name string `form:"name"`
}

type ItemsPostBody struct {
	CreatedBy string `json:"createdBy" validate:"required"`
	Name string `json:"name" validate:"required"`
	Price float32 `json:"price" validate:"required"`
	Unit string `json:"unit" validate:"required"`
}

type ItemsPutBody struct {
	PublicId string `json:"id" validate:"required"`
	Name string `json:"name"`
	Price float32 `json:"price"`
	Unit string `json:"unit"`
}

type ItemsDeleteBody struct {
	PublicId string `json:"id" validate:"required"`
}

type Item struct {
	PublicId string `db:"public_id"`
	Name string `db:"name"`
	Price float32 `db:"price"`
	Unit string `db:"unit"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ReceiptsGetQuery struct {
	PublicId string `form:"publicId"`
	CreatedBy string `form:"createdBy"`
	LocationId string `form:"locationId"`
}

type ReceiptsPostBody struct {
	LocationPublicId string `json:"locationId" validate:"required"`
	CreatedByPublicId string `json:"createdBy" validate:"required"`
}

type ReceiptId struct {
	Id int `db:"id"`
}

type Receipt struct {
	PublicId string `db:"public_id"`
	LocationId string `db:"location_id"`
	CreatedBy string `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ItemInReceipt struct {
	PublicId string `db:"public_id"`
	ItemPublicId string `db:"item_public_id"`
	Name string `db:"item_name"`
	Price float32 `db:"item_price"`
	Unit string `db:"item_unit"`
	Amount float32 `db:"amount"`
}