# Receipts Archive Backend

Backed for Receipts Archive written in Go. This is a work in progress and project that I use to loearn more about full stack development and deployment and fix everyday problems that I have.

Goal is to have an web app that would be used for storing receipts from grocery shopping and see stats about what items are bought the most, where are they bought, how much do they cost and other usefull stuff.

It's written in Go so it's very easy to deploy it with Docker.

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

You'll need to set environment variables before running the backend. You can use `.env` file to store environment variables without specifing them on every execution of the backend.

Here is the list of environment variables and their description.

|Environment variable|Description|
|-|-|
|COOKIE_SECRET|Secret for the cookie|
|JWT_KEY|Key for generating JWT|
|GOOGLE_OAUTH_CLIENT_KEY|Client id for Google oauth|
|GOOGLE_OAUTH_CLIENT_SECRET|Client secret for Google auth|
|GOOGLE_OAUTH_CALLBACK_URL|Callback url for oauth on the backend (use localhost if in dev mode)|
|AUTH_CALLBACK|Callback url to the frontend after authentication is finished (use localhost if in dev mode)|
|ALLOW_ORIGINS|Allowed origins (use localhost if in dev mode)|
|PORT|Port on which server will listen for requests|

To run the backend just run the built binary
```sh
$ ./main
```

The SQLite database will be automatically generated when running the backend for the first time.

## License
MIT
