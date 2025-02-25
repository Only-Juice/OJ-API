package database

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"OJ-API/config"
)

// DBConn is a pointer to gorm.DB
var DBConn *gorm.DB

// Connect creates a connection to database
func Connect() (err error) {
    p := config.Config("DB_PORT")
    port, err := strconv.ParseUint(p, 10, 32)
    if err != nil {
		return err
	}

    // Connection URL to connect to Postgres Database
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", config.Config("DB_HOST"), port, config.Config("DB_USER"), config.Config("DB_PASSWORD"), config.Config("DB_NAME"))
    // Connect to the DB and initialize the DB variable
    DBConn, err = gorm.Open(postgres.Open(dsn))
	if err != nil {
		return err
	}

    sqlDB, err := DBConn.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

		return nil
	}