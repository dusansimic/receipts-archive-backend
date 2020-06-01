package handlers

// ContextKey is a custom type string for context key
type ContextKey string

// StructID : Structure for getting id
type StructID struct {
	ID int `db:"id"`
}
