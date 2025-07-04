package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
	"log"
	"os"
	"strconv"
)

var dialer *gomail.Dialer

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file " + err.Error())
	}

	err = connectToDatabase(os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_DATABASE"),
	)

	if err != nil {
		log.Fatal(err)
	}

	smtpp, err := strconv.Atoi(os.Getenv("SMTP_PORT"))

	if err != nil {
		log.Fatal(err)
	}

	dialer = gomail.NewDialer(os.Getenv("SMTP_HOST"), smtpp, os.Getenv("SMTP_LOGIN"), os.Getenv("SMTP_PASSWORD"))

	fmt.Println("Bot is up")
	log.Fatal(initBot(os.Getenv("TELEGRAM_BOT_TOKEN")))
}
