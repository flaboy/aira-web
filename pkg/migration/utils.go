package migration

import (
	"github.com/flaboy/aira/aira-core/pkg/database"
	"gorm.io/gorm"
)

func AutoDropUnusedColumns() error {
	if database.Database() == nil {
		return nil
	}

	// Drop unused columns in all registered models
	for _, model := range needAutoMigrations {
		err := dropUnusedColumns(model)
		if err != nil {
			return err
		}
	}
	return nil
}

func dropUnusedColumns(dst interface{}) error {
	db := database.Database()
	stmt := &gorm.Statement{DB: db}
	stmt.Parse(dst)
	fields := stmt.Schema.Fields
	columns, _ := db.Migrator().ColumnTypes(dst)

	for i := range columns {
		found := false
		for j := range fields {
			if columns[i].Name() == fields[j].DBName {
				found = true
				break
			}
		}
		if !found {
			err := db.Debug().Migrator().DropColumn(dst, columns[i].Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

var needAutoMigrations []interface{}

func RegisterAutoMigrateModels(models ...interface{}) {
	needAutoMigrations = append(needAutoMigrations, models...)
}
