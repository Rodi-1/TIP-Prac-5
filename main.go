package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is empty; see .env.example")
	}
	db, err := openDB(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	repo := NewRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	firstID := 0
	for _, title := range []string{"Сделать ПЗ №5", "Купить кофе", "Проверить отчёты"} {
		id, err := repo.CreateTask(ctx, title)
		if err != nil {
			log.Fatal(err)
		}
		if firstID == 0 {
			firstID = id
		}
		log.Printf("Inserted task id=%d (%s)", id, title)
	}
	if err := repo.CreateMany(ctx, []string{"Массовая вставка: первая", "Массовая вставка: вторая"}); err != nil {
		log.Fatal(err)
	}
	ctxList, cancelList := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelList()
	tasks, err := repo.ListTasks(ctxList)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Tasks ===")
	for _, t := range tasks {
		fmt.Printf("#%d | %s | done=%v | %s\n", t.ID, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339))
	}
	for _, done := range []bool{false, true} {
		items, err := repo.ListDone(ctxList, done)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ListDone(%v): %d tasks\n", done, len(items))
	}
	task, err := repo.FindByID(ctxList, firstID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("FindByID(%d): title=%s, done=%v, created_at=%s\n",
		task.ID, task.Title, task.Done, task.CreatedAt.Format(time.RFC3339))
}
