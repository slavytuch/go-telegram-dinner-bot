package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file " + err.Error())
	}
	pref := tele.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	b.Handle("/start", start)
	b.Handle("/menu", showMenu)

	b.Handle(tele.OnText, handleText)

	b.Handle("/subscribe", func(c tele.Context) error {
		chatId := int(c.Chat().ID)

		chat, err := findChat(chatId)

		if err != nil {
			return err
		}

		if chat == nil {
			return c.Send("Нужно сначала зарегистрироваться в системе -> /start")
		}

		return c.Send("Inlines", &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{{
				tele.InlineButton{
					Text: "Текущая неделя",
					Data: "current-week",
				},
				tele.InlineButton{
					Text: "Следующая неделя",
					Data: "next-week",
				},
			}},
		})

	})

	b.Handle(tele.OnCallback, func(c tele.Context) error {
		fmt.Println("On callback called")
		fmt.Println(c.Callback().Data)
		var text string
		switch c.Callback().Data {
		case "current-week":
			text = "Current week"
			break
		case "next-week":
			text = "Next week"
			break
		}

		return c.Respond(&tele.CallbackResponse{
			Text: text,
		})
	})

	err = connectToDatabase(os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_DATABASE"),
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Bot is starting...")
	b.Start()
}
