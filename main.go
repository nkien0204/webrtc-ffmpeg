package main

import (
	"github.com/joho/godotenv"
	"github.com/nkien0204/lets-go/cmd"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	cmd.Execute()
}
