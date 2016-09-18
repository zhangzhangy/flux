package sql

import (
	"testing"
	"time"

	"github.com/weaveworks/fluxy"
)

func TestNew(t *testing.T) {
	if _, err := New("ql-mem", "automation"); err != nil {
		t.Fatal(err)
	}
}

func TestSchedule(t *testing.T) {
	var db *DB
	var err error
	id := flux.InstanceID("instance-123")

	if db, err = New("ql-mem", "automation"); err != nil {
		t.Fatal(err)
	}
	if err = db.ScheduleCheck(id, 0*time.Second); err != nil {
		t.Fatal(err)
	}
	check, err := db.CandidatesForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if len(check) == 1 && check[0] == flux.InstanceID(id) {
		return
	}
	t.Fatalf("Expected %#v, got %#v", []flux.InstanceID{id}, check)
}
