package main

import "shiro-email/backend/internal/bootstrap"

func main() {
	bootstrap.MustRunHTTPServer()
}
