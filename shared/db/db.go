package db

import (
	"database/sql"
	"log"
)

type Configuration struct {
	Driver         string
	DataSourceName string
}

func Open(config *Configuration) (db *sql.DB) {
	db, err := sql.Open(config.Driver, config.DataSourceName)
	if err != nil {
		panic(err.Error())
	}
	if err = db.Ping(); err != nil {
		panic(err.Error())
	}

	log.Print("[database] successfully opened")
	return
}
