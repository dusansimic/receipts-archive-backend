package resolvers

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/dusansimic/receipts-archive-backend/handlers"
	graphql "github.com/graph-gophers/graphql-go"
)

// LocationResolver is a struct for resolved location
type LocationResolver struct {
	location handlers.Location
}

// LocationResolverArgs is a struct for location resolver arguments
type LocationResolverArgs struct {
	Name *string
}

// Locations is a locations resolver. If name argument is specified, it searches
// for locations by name
func (r *Resolver) Locations(ctx context.Context, args LocationResolverArgs) (*[]*LocationResolver, error) {
	publicID := GetUserID(ctx)
	user, err := publicID.PrivateID(r.db)
	if err != nil {
		return nil, err
	}

	query := sq.Select("public_id, name, address, created_at, updated_at").From("locations").Where(sq.Eq{"created_by": user.ID})

	name := args.Name
	if name != nil {
		query = query.Where(sq.Like{"name": fmt.Sprint("%", name, "%")})
	}

	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	locations := []handlers.Location{}
	if err := r.db.Select(&locations, queryString, queryStringArgs...); err != nil {
		return nil, err
	}

	resolver := make([]*LocationResolver, 0, len(locations))
	for _, location := range locations {
		resolver = append(resolver, &LocationResolver{
			location: location,
		})
	}

	return &resolver, nil
}

// ID get the id field from location
func (r *LocationResolver) ID() string {
	return r.location.PublicID
}

// Name gets the name field from the location
func (r *LocationResolver) Name() string {
	return r.location.Name
}

// Address gets the address field from the location
func (r *LocationResolver) Address() string {
	return r.location.Address
}

// CreatedAt get the createdAt field from the location
func (r *LocationResolver) CreatedAt() graphql.Time {
	return graphql.Time{
		Time: r.location.CreatedAt,
	}
}

// UpdatedAt gets the updatedAt field from the location
func (r *LocationResolver) UpdatedAt() graphql.Time {
	return graphql.Time{
		Time: r.location.UpdatedAt,
	}
}
