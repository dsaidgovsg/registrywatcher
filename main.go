package main

import (
	"fmt"

	"registrywatcher/app/repository"
	"registrywatcher/server"
	"registrywatcher/utilities/config"
	"registrywatcher/utilities/db"
)

func main() {
	c := config.InitConfig(config.GetEnv())

	db := db.InitDB(c.DSN)

	repoService := repository.RepositoryService{
		DB: db,
	}

	if err := server.InitRoutes(repoService).Run(); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v\n", err))
	}
}
