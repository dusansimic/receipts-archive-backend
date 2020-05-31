module github.com/dusansimic/receipts-archive-backend

go 1.14

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/friendsofgo/graphiql v0.2.2
	github.com/gbrlsnchs/jwt/v3 v3.0.0-rc.2
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/sessions v0.0.3
	github.com/gin-gonic/gin v1.6.3
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/graph-gophers/graphql-go v0.0.0-20200309224638-dae41bde9ef9
	github.com/jkomyno/nanoid v0.0.0-20170914145641-30c81465692e
	github.com/jmoiron/sqlx v1.2.0
	github.com/joho/godotenv v1.3.0
	github.com/markbates/goth v1.64.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
)

replace github.com/graph-gophers/graphql-go v0.0.0-20200309224638-dae41bde9ef9 => github.com/dusansimic/graphql-go v0.0.0-20200527085124-d2e01d8becaa
