# Receipts Archive Backend

Backed for Receipts Archive written in Go. This is a work in progress and project that I use to learn more about full stack development and deployment and fix everyday problems that I have.

Goal is to have an web app that would be used for storing receipts from grocery shopping and see stats about what items are bought the most, where are they bought, how much do they cost and other useful stuff.

It's written in Go so it's very easy to deploy it with Docker or other containerized environments.

## Build

This is worked on and tested on a Linux machine so these instructions are confirmed to work for Linux.

To install all dependencies run
```sh
$ go mod download
```

and build the backend run
```sh
$ go build -o main
```

## Run

You'll need to set environment variables before running the backend. You can use `.env` file to store environment variables without specifying them on every execution of the backend.

Here is the list of environment variables and their description.

|Environment variable|Description|
|-|-|
|SESSION_COOKIE_SECRET|Secret for the session cookie|
|GOTHIC_COOKIE_SECRET|Secret for the Gothic cookie (3rd party service OAuth)|
|GOOGLE_OAUTH_CLIENT_KEY|Client id for Google oauth|
|GOOGLE_OAUTH_CLIENT_SECRET|Client secret for Google auth|
|GOOGLE_OAUTH_CALLBACK_URL|Callback url for oauth on the backend (use localhost if in dev mode)|
|AUTH_CALLBACK|Callback url to the frontend after authentication is finished (use localhost if in dev mode)|
|ALLOW_ORIGINS|Allowed origins (use localhost if in dev mode)|
|SESSION_STORE_DATABASE_ADDRESS|Address of the Memcached database used for storing session data|
|PORT|Port on which server will listen for requests|

To run the backend just run the built binary
```sh
$ ./main
```

The SQLite database will be automatically generated when running the backend for the first time.

### Docker

You can also run this backend inside a docker container. Just pull the image and run it with this command.

```sh
$ docker pull dusansimic/receipts-archive-backend
$ docker run -e ... dusansimic/receipts-archive-backend
```

Other options is to use `.env` file with the following command.

```sh
$ docker run --env-file .env dusansimic/receipts-archive-backend
```

## Deployment

If you want to deploy this with frontend, you should check [this](https://gitlab.com/makerns/receipts-archive/deployment) out.

## License
MIT
