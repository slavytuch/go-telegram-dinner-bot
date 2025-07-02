package main

import (
	"fmt"
	tele "gopkg.in/telebot.v4"
	"net/mail"
)

func showMenu(c tele.Context) error {
	menu, err := getMenu()

	if err != nil {
		return err
	}

	return c.Send(menu.Menu)
}

func start(c tele.Context) error {
	chat, err := findChat(int(c.Chat().ID))

	if err != nil {
		return err
	}

	if chat == nil {
		c.Send("Вы запустили бот обедов на golang. Пожайлуста введите рабочий email")
		return nil
	}

	if chat.Fio == nil {
		return c.Send("Вы успешно зарегистрировались и теперь можете записаться на обед. Но прежде, введи Фамилию и Имя в формате 'Иванов Иван'")
	}

	return c.Send(randomGreet() + ", " + *chat.Fio)
}

func handleText(c tele.Context) error {
	chatId := int(c.Chat().ID)
	chat, err := findChat(chatId)

	if err != nil {
		return err
	}

	if chat == nil {
		if !CodeSent(chatId) {
			_, err = mail.ParseAddress(c.Text())

			if err == nil {
				err = SendVerificationCode(c.Text(), chatId)
				if err != nil {
					c.Send("Внутренняя ошибка")
					panic(err)
				}

				return c.Send("Проверьте почту, и введите код из письма")
			}

			return start(c)
		}

		if CheckVerificationCode(chatId, c.Text()) {
			err = CreateChat(chatId, c.Chat().Title, GetEmailByChat(chatId))
			if err != nil {
				c.Send("Внутренняя ошибка")
				panic(err)
			}
			return c.Send("Вы успешно зарегистрировались и теперь можете записаться на обед. Но прежде, введи Фамилию и Имя в формате 'Иванов Иван'")
		} else {
			return c.Send("Неверный код")
		}
	}

	if chat.Fio == nil {
		err = SetChatFio(chatId, c.Text())
		if err != nil {
			c.Send("Внутренняя ошибка")
			panic(err)
		}
		return c.Send(c.Text() + ", спасибо, теперь можете пользоваться ботом")
	} else {
		return start(c)
	}

	return nil
}

func currentWeek(c tele.Context) error {
	fmt.Println("Current week called")
	return c.Respond(&tele.CallbackResponse{
		Text: "Current week",
	})
}

func nextWeek(c tele.Context) error {
	fmt.Println("Next week called")
	return c.Respond(&tele.CallbackResponse{
		Text: "Next week",
	})
}
