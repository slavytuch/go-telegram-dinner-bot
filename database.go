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
	Id       int
	ChatId   int
	Date     time.Time
	Status   int
	FIO      string
	ParentId sql.NullInt64
}

func (s *Subscription) parse(row *sql.Rows) error {
	defer row.Close()
	return row.Scan(&s.Id, &s.ChatId, &s.Date, &s.Status, &s.FIO, &s.ParentId)
}

func (s *Subscription) parseRow(row *sql.Row) error {
	return row.Scan(&s.Id, &s.ChatId, &s.Date, &s.Status, &s.FIO, &s.ParentId)
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
func subscriptionSelect() string {
	return "SELECT subscribes.id, telegraph_chats.chat_id, subscribes.date, subscribes.status, telegraph_chats.fio, subscribes.parent_id FROM subscribes INNER JOIN telegraph_chats ON subscribes.telegraph_chats_id = telegraph_chats.id"
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

func findSubscriptionByChat(chatId int, date time.Time) (*Subscription, error) {
	var s Subscription

	err := s.parseRow(db.QueryRow(subscriptionSelect()+" where telegraph_chats.chat_id = ? and date = ?", chatId, date))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &s, nil
}

func findSubscriptionById(id int) (*Subscription, error) {
	var s Subscription

	err := s.parseRow(db.QueryRow(subscriptionSelect()+" where subscribes.id = ?", id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &s, nil
}

func findBookingSubscription(id int) (*Subscription, error) {
	var s Subscription

	err := s.parseRow(db.QueryRow(subscriptionSelect()+" where subscribes.parent_id = ?", id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &s, nil
}

func findSubscriptionList(chatId int, ds time.Time, de time.Time) ([]Subscription, error) {
	rows, err := db.Query(subscriptionSelect()+" where telegraph_chats.chat_id = ? and date >= ? and date <= ?", chatId, ds, de)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subs []Subscription

	for rows.Next() {
		var sub Subscription

		if err := sub.parse(rows); err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func findRefusedSubscriptionList(ds time.Time, de time.Time, c int) ([]Subscription, error) {
	rows, err := db.Query(subscriptionSelect()+" LEFT JOIN subscribes booked on subscribes.id = booked.parent_id LEFT JOIN telegraph_chats btc on booked.telegraph_chats_id = telegraph_chats.id WHERE subscribes.status = ? and subscribes.date >= ? and subscribes.date <= ? or (subscribes.status = ? and btc.chat_id = ?)", Refuse, ds, de, Booked, c)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subs []Subscription

	for rows.Next() {
		var sub Subscription

		if err := sub.parse(rows); err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func findSubscriptionCount(ds time.Time, de time.Time) (map[string]int, error) {
	rows, err := db.Query("SELECT COUNT(*), date FROM subscribes WHERE date >= ? && date <= ? GROUP BY date", ds, de)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make(map[string]int)
	var count int
	var date time.Time

	for rows.Next() {
		if err := rows.Scan(&count, &date); err != nil {
			return nil, err
		}

		result[date.Format("0201")] = count
	}

	return result, nil
}

func finSubscriptionsByDay(d time.Time) ([]string, error) {
	var result []string

	rows, err := db.Query("SELECT telegraph_chats.fio FROM telegraph_chats INNER JOIN subscribes ON subscribes.telegraph_chats_id = telegraph_chats.id WHERE date = ?", d)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var name string
	for rows.Next() {
		rows.Scan(&name)

		result = append(result, name)
	}

	return result, nil
}

func createSubscription(chatId int, date time.Time, s int) error {
	chat, err := findChat(chatId)

	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO subscribes(telegraph_chats_id, date, status) VALUES (?, ?, ?)", chat.Id, date, s)

	return err
}

func createBookingSubscription(chatId int, date time.Time, s int, p int) error {
	chat, err := findChat(chatId)

	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO subscribes(telegraph_chats_id, date, status, parent_id) VALUES (?, ?, ?, ?)", chat.Id, date, s, p)

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

func updateMenu(menu string) error {
	_, err := db.Exec("UPDATE menu SET menu = ?", menu)

	return err
}

func getAllChats() ([]int, error) {
	var result []int

	rows, err := db.Query("SELECT id from telegraph_chats")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var chatId int

		err := rows.Scan(&chatId)

		if err != nil {
			return nil, err
		}

		result = append(result, chatId)
	}

	return result, nil
}
