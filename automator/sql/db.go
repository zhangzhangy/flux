package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/weaveworks/fluxy"
)

type DB struct {
	conn *sql.DB
}

func New(driver, source string) (*DB, error) {
	conn, err := sql.Open(driver, source)
	if err != nil {
		return nil, err
	}
	tx, err := conn.Begin()
	if err == nil {
		_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS automation_schedule
                          (instance string NOT NULL,
                           next_check time NOT NULL)`)
		if err == nil {
			err = tx.Commit()
		}
	}
	if err != nil {
		return nil, err
	}
	return &DB{conn: conn}, nil
}

func (db *DB) Remove(instance flux.InstanceID) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM automation_schedule WHERE instance = $1`, string(instance)); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) CandidatesForUpdate() ([]flux.InstanceID, error) {
	rows, err := db.conn.Query(`SELECT instance FROM automation_schedule WHERE next_check < now()`)
	if err != nil {
		return nil, err
	}
	candidates := []flux.InstanceID{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		candidates = append(candidates, flux.InstanceID(id))
	}
	return candidates, rows.Err()
}

func (db *DB) ScheduleCheck(id flux.InstanceID, after time.Duration) error {
	tx, err := db.conn.Begin()
	if err == nil {
		_, err = tx.Exec(`DELETE FROM automation_schedule WHERE instance = $1`, string(id))
		if err == nil {
			_, err = tx.Exec(`INSERT INTO automation_schedule
                                   (instance, next_check) VALUES ($1, now() + duration($2))`,
				string(id), fmt.Sprintf("%ds", after/time.Second))
			if err == nil {
				err = tx.Commit()
			}
		}
	}
	return err
}
