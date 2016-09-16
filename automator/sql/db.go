package sql

import (
	"database/sql"
	"time"

	"github.com/weaveworks/fluxy"
)

type DB struct {
	conn sql.DB
}

func New(driver, source string) (*DB, error) {
	return &DB{}, nil
}

func (db *DB) CandidatesForUpdate() ([]flux.InstanceID, error) {
	return []flux.Instance{}, nil
}

func (db *DB) MarkUnderway(id flux.InstanceID, timeout time.Duration) error {
	return nil
}

func (db *DB) ScheduleCheck(id flux.InstanceID, after time.Duration) error {
	return nil
}
