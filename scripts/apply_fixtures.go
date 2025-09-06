package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Получаем строку подключения к БД из переменной окружения
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/toloko?sslmode=disable"
	}

	// Подключаемся к БД
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatal("Ошибка ping БД:", err)
	}

	// Читаем SQL файл с фикстурами
	sqlFile, err := ioutil.ReadFile("migrations/001_add_fixtures.sql")
	if err != nil {
		log.Fatal("Ошибка чтения SQL файла:", err)
	}

	// Выполняем SQL
	_, err = db.Exec(string(sqlFile))
	if err != nil {
		log.Fatal("Ошибка выполнения SQL:", err)
	}

	fmt.Println("Фикстуры успешно добавлены в БД!")
}
