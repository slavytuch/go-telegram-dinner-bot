package main

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"time"
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

type Subscription struct {
	Id     int
	ChatId int
	Date   time.Time
	Status int
}

const (
	Active = iota
	Refuse
	Booked
)

func connectToDatabase(host string, port string, user string, password string, database string) error {
	dbConfig := mysql.NewConfig()

	dbConfig.Addr = host + ":" + port
	dbConfig.User = user
	dbConfig.Passwd = password
	dbConfig.Net = "tcp"
	dbConfig.DBName = database
	dbConfig.ParseTime = true

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

	return err
}

func findSubscription(chatId int, date time.Time) (*Subscription, error) {
	var s Subscription

	err := db.QueryRow("SELECT subscribes.id, telegraph_chats.chat_id, date, status FROM subscribes INNER JOIN telegraph_chats ON subscribes.telegraph_chats_id = telegraph_chats.id where telegraph_chats.chat_id = ? and date = ?", chatId, date).Scan(&s.Id, &s.ChatId, &s.Date, &s.Status)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &s, nil
}

func findSubscriptionList(chatId int, ds time.Time, de time.Time) ([]Subscription, error) {
	rows, err := db.Query("SELECT subscribes.id, telegraph_chats.chat_id, date, status FROM subscribes INNER JOIN telegraph_chats ON subscribes.telegraph_chats_id = telegraph_chats.id where telegraph_chats.chat_id = ? and date >= ? and date <= ?", chatId, ds, de)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subs []Subscription

	for rows.Next() {
		var sub Subscription

		if err := rows.Scan(&sub.Id, &sub.ChatId, &sub.Date, &sub.Status); err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func createSubscription(chatId int, date time.Time, s int) error {
	chat, err := findChat(chatId)

	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO subscribes(telegraph_chats_id, date, status) VALUES (?, ?, ?)", chat.Id, date, s)

	return err
}

func updateSubscriptionStatus(id int, st int) error {
	_, err := db.Exec("UPDATE subscribes SET status = ? WHERE id = ?", st, id)

	return err
}

func deleteSubscription(id int) error {
	_, err := db.Exec("DELETE FROM subscribes WHERE id = ?", id)

	return err
}
