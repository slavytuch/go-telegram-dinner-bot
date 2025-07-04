package main

import (
	"github.com/jinzhu/now"
	tele "gopkg.in/telebot.v4"
	"net/mail"
	"strings"
	"time"
)

const (
	inlineCurrentWeek = "current-week"
	inlineNextWeek    = "next-week"
	inlineToggleSub   = "toggle-subscription"
	inlineBackToList  = "back-to-list"
)

func initBot(t string) error {
	pref := tele.Settings{
		Token:  t,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	keyboardWeeks := tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{{
			tele.InlineButton{
				Text: "Текущая неделя",
				Data: inlineCurrentWeek,
			},
			tele.InlineButton{
				Text: "Следующая неделя",
				Data: inlineNextWeek,
			},
		}},
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

		return c.Send("Проставьте, дни обеда.\n"+
			"Записаться на следующий день можно до 12:00!\n"+
			"Запись и отмена записи проиводится кликом по кнопке с днем недели.\n"+
			"Если вы записались на обед и он уже заказан, но вы не придете - кликните на этот день, мы учтем освободившийся обед, а у вас он отобразиться значком - ⚠️\n"+
			"Статус ⚠️ изменить нельзя, возможно ваш обед уже съели. Пишите в тех. поддержку.\n\n"+
			"❌ - Не приду\n"+
			"✅ - Заказать обед\n"+
			"⚠️ - Вы записаны, но не придете", &keyboardWeeks)
	})

	b.Handle(tele.OnCallback, func(c tele.Context) error {
		data := c.Callback().Data

		if len(data) > 0 && data[0] == '\f' {
			data = data[1:]
		}

		unique := data

		split := strings.Split(data, "|")
		if len(split) == 2 {
			unique = split[0]
			data = split[1]
		}

		switch unique {
		case inlineCurrentWeek:
			return replaceWeekButtons(c, now.Monday())
		case inlineNextWeek:
			return replaceWeekButtons(c, now.With(time.Now().AddDate(0, 0, 7)).Monday())
		case inlineToggleSub:
			pd, err := time.Parse("20060102", data)
			if err != nil {
				return err
			}

			return toggleSubscription(c, pd)
		case inlineBackToList:
			c.Respond()

			return c.Edit(c.Message().Text, &keyboardWeeks)
		}

		return c.Respond(&tele.CallbackResponse{
			Text: "Тут могла быть ваша реклама",
		})
	})

	b.Start()
	return nil
}

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
		clearVerificationCode(int(c.Chat().ID))
		c.Send("Вы запустили бот обедов на golang. Пожайлуста введите рабочий email")
		return nil
	}

	if chat.Fio == nil {
		return c.Send("Вы успешно зарегистрировались и теперь можете записаться на обед. Но прежде, введи Фамилию и Имя в формате 'Иванов Иван'")
	}

	return c.Send(randomGreet() + ", " + *chat.Fio)
}

var timer *time.Timer

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

				timer = time.AfterFunc(15*time.Second, func() {
					clearVerificationCode(chatId)
					c.Send("Время подтверждения email истекло, прошу запустить процедуру ещё раз -> /start")
				})

				return c.Send("Проверьте почту, и введите код из письма")
			}

			return start(c)
		}

		if CheckVerificationCode(chatId, c.Text()) {
			if timer != nil {
				timer.Stop()
			}
			err = CreateChat(chatId, c.Chat().Title, GetEmailByChat(chatId))
			clearVerificationCode(chatId)
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
		return c.Send(c.Text() + ", спасибо, теперь можете пользоваться ботом -> /subscribe")
	} else {
		return start(c)
	}

	return nil
}

func replaceWeekButtons(c tele.Context, ds time.Time) error {
	var buttonList []tele.Row
	sl, err := findSubscriptionList(int(c.Chat().ID), ds, ds.AddDate(0, 0, 5))

	if err != nil {
		return err
	}

	sortedList := make(map[string]Subscription)
	for _, sub := range sl {
		sortedList[sub.Date.Format("0201")] = sub
	}

	m := &tele.ReplyMarkup{}

	for ds.Weekday() != time.Saturday {
		icon := "✅"
		sub, ok := sortedList[ds.Format("0201")]
		if !ok {
			icon = "❌"
		} else if sub.Status == Refuse && time.Now().Day() == ds.Day() {
			icon = "⚠️"
		}

		buttonList = append(buttonList, m.Row(m.Data(icon+" "+[]string{
			"Воскресенье",
			"Понедельник",
			"Вторник",
			"Среда",
			"Четверг",
			"Пятница",
			"Суббота",
		}[ds.Weekday()]+", "+ds.Format("02")+" "+[]string{
			"Января", "Февраля", "Марта", "Апреля", "Мая", "Июня", "Июля", "Августа", "Сентября", "Октября", "Ноября", "Декабря",
		}[ds.Month()], inlineToggleSub, ds.Format("20060102"))))

		ds = ds.Add(time.Hour * 24)
	}
	buttonList = append(buttonList, m.Row(m.Data("Назад", inlineBackToList)))

	m.Inline(buttonList...)
	return c.Edit(c.Message().Text, m)
}

func toggleSubscription(c tele.Context, t time.Time) error {
	if now.BeginningOfDay().After(t) {
		return c.Respond(&tele.CallbackResponse{
			Text: "Не, менять запись на прошлые дни неззя",
		})
	}

	isLate := time.Now().Hour() >= 12 && time.Now().AddDate(0, 0, 1).After(t)

	sub, err := findSubscription(int(c.Chat().ID), t)

	if err != nil {
		c.Respond(&tele.CallbackResponse{
			Text: "Йа СлОмАлСо!",
		})
		return err
	}

	if sub == nil {
		if time.Now().Day() == t.Day() {
			return c.Respond(&tele.CallbackResponse{
				Text: "Записываться на сегодня неззя",
			})
		}

		if isLate {
			return c.Respond(&tele.CallbackResponse{
				Text: "Запись на завтра закрыта :(",
			})
		}

		err = createSubscription(int(c.Chat().ID), t, Active)

		if err != nil {
			c.Respond(&tele.CallbackResponse{
				Text: "Йа СлОмАлСо!",
			})

			return err
		}

		c.Respond(&tele.CallbackResponse{
			Text: "Запись на " + t.Format("02.01") + " успешна",
		})

		return replaceWeekButtons(c, now.With(t).Monday())
	}

	if sub.Status == Booked {
		return c.Respond(&tele.CallbackResponse{
			Text: "Запись на " + t.Format("02.01") + " уже кто-то забронировал",
		})
	}

	if isLate {
		if sub.Status == Refuse {
			return c.Respond(&tele.CallbackResponse{
				Text: "Запись на " + t.Format("02.01") + " уже нельзя поменять - слишком поздно",
			})
		}

		err = updateSubscriptionStatus(sub.Id, Refuse)

		if err != nil {
			c.Respond(&tele.CallbackResponse{
				Text: "Йа СлОмАлСо!",
			})

			return err
		}

		c.Respond(&tele.CallbackResponse{
			Text: "Запись на " + t.Format("02.01") + " помечена как \"Отказался\"",
		})

		return replaceWeekButtons(c, now.With(t).Monday())
	}

	err = deleteSubscription(sub.Id)

	if err != nil {
		c.Respond(&tele.CallbackResponse{
			Text: "Йа СлОмАлСо!",
		})

		return err
	}

	c.Respond(&tele.CallbackResponse{
		Text: "Запись на " + t.Format("02.01") + " удалена",
	})

	return replaceWeekButtons(c, now.With(t).Monday())
}
