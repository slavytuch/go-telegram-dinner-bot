package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
)

var db *sql.DB

type Menu struct {
	Menu string
}

type Chat struct {
	Id     int
	ChatId int
	Name   string
	Email  string
	Fio    *string
}

func connectToDatabase(host string, port string, user string, password string, database string) error {
	dbConfig := mysql.NewConfig()

	dbConfig.Addr = host + ":" + port
	dbConfig.User = user
	dbConfig.Passwd = password
	dbConfig.Net = "tcp"
	dbConfig.DBName = database

	var err error
	db, err = sql.Open("mysql", dbConfig.FormatDSN())

	if err != nil {
		return err
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return pingErr
	}

	return nil
}

func getMenu() (*Menu, error) {
	var menu Menu

	err := db.QueryRow("SELECT menu FROM menu").Scan(&menu.Menu)

	if err != nil {
		return nil, err
	}

	return &menu, nil
}

func findChat(chatId int) (*Chat, error) {
	var chat Chat

	err := db.QueryRow("SELECT id, chat_id, name, email, fio FROM telegraph_chats WHERE chat_id = ?", chatId).
		Scan(&chat.Id, &chat.ChatId, &chat.Name, &chat.Email, &chat.Fio)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &chat, nil
}

func CreateChat(chatId int, chatName string, email string) error {
	_, err := db.Exec("INSERT INTO telegraph_chats (chat_id, name, email, telegraph_bot_id) VALUES (?, ?, ?, 1)", chatId, chatName, email)

	return err
}

func SetChatFio(chatId int, chatFio string) error {
	_, err := db.Exec("UPDATE telegraph_chats SET fio=? WHERE chat_id = ?", chatFio, chatId)

	fmt.Println(err)
	return err
}
