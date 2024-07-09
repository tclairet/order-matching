package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	db, err := newPostgres(os.Getenv("DB_URL"))
	if err != nil {
		panic(err)
	}
	seed(db)

	api := newAPI(db)

	port := "8080"
	slog.Info("listening", "port", port)

	// todo do gracefully shut down
	// https://pkg.go.dev/net/http#Server.Shutdown
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), api.routes()))
}
