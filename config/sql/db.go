package sql

import (
	"database/sql"
	"os"
	//	"encoding/json"

	"github.com/pkg/errors"

	"github.com/weaveworks/fluxy/config"

	_ "github.com/cznic/ql/driver"
	_ "github.com/lib/pq"
)

var (
	ErrNoSchemaDefinedForDriver = errors.New("schema not defined for driver")

	qlSchema = `
      CREATE TABLE IF NOT EXISTS config
        (instance string NOT NULL
         config   string NOT NULL
         stamp    time NOT NULL)
    `

	pgSchema = `
      CREATE TABLE IF NOT EXISTS config
        (instance varchar(255) NOT NULL,
         config   text NOT NULL,
         stamp    timestamp with time zone NOT NULL)
    `

	schemaByDriver = map[string]string{
		"ql":       qlSchema,
		"ql-mem":   qlSchema,
		"postgres": pgSchema,
	}
)

type db struct {
	conn   *sql.DB
	schema string
}

func New(driver, datasource string) (*db, error) {
	conn, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, err
	}
	db := &db{
		conn:   conn,
		schema: schemaByDriver[driver],
	}
	if db.schema == "" {
		return nil, ErrNoSchemaDefinedForDriver
	}
	return db, db.ensureTables()
}

func (db *db) Set(instance config.InstanceID, conf config.InstanceConfig) error {
	return nil
}

func (db *db) Get(instance config.InstanceID) (config.InstanceConfig, error) {
	return config.InstanceConfig, nil
}

// ---

func (db *db) ensureTables() error {
	// ql driver needs this to work correctly in a container
	os.MkDir(os.TempDir(), 0777)
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(db.schema)
	if err != nil {
		return err
	}
	return tx.Commit()
}
