module github.com/Frostauk/caddy-exec

// Taken from: https://stackoverflow.com/a/60343127
replace github.com/abiosoft/caddy-exec => github.com/Frostauk/caddy-exec

go 1.14

require (
	github.com/caddyserver/caddy/v2 v2.4.1
	go.uber.org/zap v1.16.0
)
