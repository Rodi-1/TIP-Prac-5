package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"
)

func testDB(t testing.TB) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL for PostgreSQL integration checks")
	}
	db, err := openDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRepository(t *testing.T) {
	db := testDB(t)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `CREATE TEMP TABLE tasks (
		id SERIAL PRIMARY KEY, title TEXT NOT NULL CHECK (title <> 'reject'),
		done BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	r := NewRepo(db)
	id, err := r.CreateTask(ctx, "SQL ' quote; DROP TABLE tasks;")
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByID(ctx, id)
	if err != nil || got.Title != "SQL ' quote; DROP TABLE tasks;" {
		t.Fatalf("parameterized insert/read: %+v, %v", got, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET done = $1 WHERE id = $2`, true, id); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateMany(ctx, []string{"batch one", "batch two"}); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateMany(ctx, []string{"must roll back", "reject"}); err == nil {
		t.Fatal("expected constraint error")
	}
	all, err := r.ListTasks(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("transaction rollback: got %d tasks, err=%v", len(all), err)
	}
	for done, want := range map[bool]int{true: 1, false: 2} {
		items, err := r.ListDone(ctx, done)
		if err != nil || len(items) != want {
			t.Fatalf("ListDone(%v): got %d, want %d, err=%v", done, len(items), want, err)
		}
	}
	if _, err := r.FindByID(ctx, 999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing id: %v", err)
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := r.ListTasks(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request: %v", err)
	}
}

func BenchmarkPool(b *testing.B) {
	for _, setting := range []struct {
		name       string
		open, idle int
	}{{"open1_idle1", 1, 1}, {"open5_idle2", 5, 2}, {"open10_idle5", 10, 5}} {
		b.Run(setting.name, func(b *testing.B) {
			db := testDB(b)
			db.SetMaxOpenConns(setting.open)
			db.SetMaxIdleConns(setting.idle)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					var count int
					err := db.QueryRowContext(ctx, `SELECT count(*) FROM tasks`).Scan(&count)
					cancel()
					if err != nil {
						b.Error(err)
					}
				}
			})
			b.StopTimer()
			b.ReportMetric(float64(db.Stats().WaitCount), "waits")
		})
	}
}
