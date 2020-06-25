package database

import (
	"os"

	// DB stuff
	"github.com/jmoiron/sqlx"
)

type SQLOptions struct {
	DatabasePath string
}

func (o SQLOptions) GenerateDatabase() (*sqlx.DB, error) {
	userTableSchema := `
	create table users (
		id integer primary key autoincrement unique,
		public_id text not null unique,
		real_name text not null
	);`
	locationsTableSchema := `
	create table locations (
		id integer primary key autoincrement unique,
		created_by integer not null,
		public_id text not null unique,
		name text not null unique,
		address text not null,
		created_at datetime default current_timestamp,
		updated_at datetime default current_timestamp,
	
		foreign key (created_by) references users(id)
	);`
	receiptsTableSchema := `
	create table receipts (
		id integer primary key autoincrement unique,
		location_id integer not null,
		created_by integer not null,
		public_id text not null unique,
		created_at datetime default current_timestamp,
		updated_at datetime default current_timestamp,

		foreign key (location_id) references locations(id),
		foreign key (created_by) references users(id)
	);`
	itemsTableSchema := `
	create table items (
		id integer primary key autoincrement unique,
		created_by integer not null,
		public_id text not null unique,
		name text not null unique,
		price real not null,
		unit text not null,
		created_at datetime default current_timestamp,
		updated_at datetime default current_timestamp,

		foreign key (created_by) references users(id)
	);`
	itemsInReceiptTableSchema := `
	create table items_in_receipt (
		id integer primary key autoincrement unique,
		receipt_id integer not null,
		item_id integer not null,
		public_id text not null unique,
		amount real default 1.0,

		foreign key (receipt_id) references receipts(id),
		foreign key (item_id) references items(id)
	);`

	if _, err := os.Stat(o.DatabasePath); err != nil {
		os.Create(o.DatabasePath)

		db, err := sqlx.Connect("sqlite3", o.DatabasePath)
		if err != nil {
			return nil, err
		}

		if _, err := db.Exec(userTableSchema); err != nil {
			return nil, err
		}
		if _, err := db.Exec(locationsTableSchema); err != nil {
			return nil, err
		}
		if _, err := db.Exec(receiptsTableSchema); err != nil {
			return nil, err
		}
		if _, err := db.Exec(itemsTableSchema); err != nil {
			return nil, err
		}
		if _, err := db.Exec(itemsInReceiptTableSchema); err != nil {
			return nil, err
		}

		return db, nil
	}

	db, err := sqlx.Connect("sqlite3", o.DatabasePath)
	if err != nil {
		return nil, err
	}

	return db, nil
}
