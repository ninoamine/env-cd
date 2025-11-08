package controller

import (
	"fmt"
	"os"
)

type PostgresqlConfiguration struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func LoadPostgresqlConfiguration() (*PostgresqlConfiguration, error) {
	cfg := &PostgresqlConfiguration{
		Host:     os.Getenv("Postgresql_Host"),
		Port:     os.Getenv("Postgresql_Port"),
		User:     os.Getenv("Postgresql_User"),
		Password: os.Getenv("Postgresql_Password"),
		Database: os.Getenv("Postgresql_Database"),
	}
	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.Password == "" || cfg.Database == "" {
		return nil, fmt.Errorf("Missing required Postgresql configuration")
	}
	return cfg, nil
}
