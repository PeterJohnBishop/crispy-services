package main

import (
	"fmt"
	"log"
	"websocket/server"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("studious-waffle is a Go(app)")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	server.ServeGin()
}
