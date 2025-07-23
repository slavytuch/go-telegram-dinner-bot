package main

import (
	"github.com/jinzhu/now"
	tele "gopkg.in/telebot.v4"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

const (
	inlineCurrentWeek = "current-week"
	inlineNextWeek    = "next-week"
	inlineToggleSub   = "toggle-subscription"
	inlineBackToList  = "back-to-list"
	inlineSubList     = "sub-list"
	inlineBook        = "book"
	inlineConfirmMenu = "confirmMenu"
	inlineDenyMenu    = "denyMenu"
)

var b *tele.Bot
var listeningForMenu map[int]bool
var newMenu string
var debug bool

func initBot(t string, d bool) error {
	pref := tele.Settings{
		Token:  t,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}
	debug = d
	listeningForMenu = make(map[int]bool)
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

	b.Handle("/list", func(c tele.Context) error {
		var buttonList []tele.Row

		ds := now.BeginningOfWeek()
		de := now.EndOfWeek().AddDate(0, 0, 7)

		countList, err := findSubscriptionCount(ds, de)
		if err != nil {
			return err
		}

		m := &tele.ReplyMarkup{}

		for ds.Day() != de.Day() {
			if ds.Weekday() == time.Saturday || ds.Weekday() == time.Sunday {
				ds = ds.Add(time.Hour * 24)
				continue
			}

			count := 0
			dbCount, ok := countList[ds.Format("0201")]
			if ok {
				count = dbCount
			}

			buttonList = append(buttonList, m.Row(m.Data([]string{
				"Воскресенье",
				"Понедельник",
				"Вторник",
				"Среда",
				"Четверг",
				"Пятница",
				"Суббота",
			}[ds.Weekday()]+", "+ds.Format("02")+" "+[]string{
				"Января", "Февраля", "Марта", "Апреля", "Мая", "Июня", "Июля", "Августа", "Сентября", "Октября", "Ноября", "Декабря",
			}[ds.Month()]+" - "+strconv.Itoa(count), inlineSubList, ds.Format("20060102"))))

			ds = ds.Add(time.Hour * 24)
		}

		m.Inline(buttonList...)
		return c.Send("Выберете день, для просмотра списка", m)
	})

	b.Handle("/booking", func(c tele.Context) error {
		subList, err := findRefusedSubscriptionList(now.BeginningOfDay(), now.EndOfDay(), int(c.Chat().ID))
		if err != nil {
			return err
		}

		if len(subList) <= 0 {
			return c.Send("❌ Свободных обедов пока нет ❌\n" +
				"Если вы знаете, что кто-то не пришел и не отметил это в боте, пожалуйста, сообщите @DariaRazan. Возможно, обед все же освободится! 🍽😊")
		}

		m, err := createBookingMarkup(subList)

		if err != nil {
			return err
		}

		return c.Send("🍽 Свободные обеды на сегодня!\n"+
			"Вот список обедов, от которых отказались. Вы можете забронировать один из них и насладиться вкусным обедом без угрызений совести! 😋\n"+
			"Приятного аппетита! 🍴", m)
	})

	b.Handle("/setmenu", func(c tele.Context) error {
		listeningForMenu[int(c.Chat().ID)] = true

		return c.Send("Жду в следующем сообщении меню. Для отмены - отправь \"отмена\"")
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
		case inlineSubList:
			pd, err := time.Parse("20060102", data)
			if err != nil {
				return err
			}

			l, err := finSubscriptionsByDay(pd)

			if len(l) <= 0 {
				return c.Respond(&tele.CallbackResponse{
					Text: "Ничего нет",
				})
			}

			var result []string
			counter := 1
			for _, name := range l {
				result = append(result, strconv.Itoa(counter)+". "+name)
				counter++
			}

			c.Respond()
			return c.Send(
				"Список записавшихся на " + pd.Format("02.01") + ":\n" +
					strings.Join(result, "\n"))
		case inlineBook:
			pr, err := strconv.Atoi(data)
			if err != nil {
				return err
			}

			sub, err := findSubscriptionById(pr)

			if err != nil {
				return err
			}

			if sub == nil {
				return c.Respond(&tele.CallbackResponse{
					Text: "Не могу найти подписку",
				})
			}

			if sub.Date.Day() != now.BeginningOfDay().Day() {
				return c.Respond(&tele.CallbackResponse{
					Text: "Забронировать за прошлые дни нельзя",
				})
			}

			switch sub.Status {
			case Active:
				return c.Respond(&tele.CallbackResponse{
					Text: "Похоже что запись больше нельзя забронировать",
				})
			case Refuse:
				userSub, err := findSubscriptionByChat(int(c.Chat().ID), now.BeginningOfDay())

				if err != nil {
					return err
				}

				if userSub == nil {
					break
				}

				if !userSub.ParentId.Valid {
					return c.Respond(&tele.CallbackResponse{
						Text: "У вас уже есть подписка на сегодня",
					})
				}

				err = deleteSubscription(userSub.Id)

				if err != nil {
					return err
				}

				break
			case Booked:
				bookedSub, err := findBookingSubscription(sub.Id)
				if err != nil {
					return err
				}

				if bookedSub == nil {
					break
				}

				if bookedSub.ChatId != int(c.Chat().ID) {
					return c.Respond(&tele.CallbackResponse{
						Text: "Запись уже забронировал " + bookedSub.FIO,
					})
				}

				err = updateSubscriptionStatus(sub.Id, Refuse)

				if err != nil {
					return err
				}

				err = deleteSubscription(bookedSub.Id)
				if err != nil {
					return err
				}
				err = replaceBookingButtons(c)
				if err != nil {
					return err
				}

				return c.Respond(&tele.CallbackResponse{
					Text: "Бронь удалена",
				})
			}

			err = updateSubscriptionStatus(sub.Id, Booked)

			if err != nil {
				return err
			}

			err = createBookingSubscription(sub.ChatId, sub.Date, Active, sub.Id)

			if err != nil {
				return err
			}

			err = replaceBookingButtons(c)
			if err != nil {
				return err
			}

			return c.Respond(&tele.CallbackResponse{
				Text: "Бронь успешно оформлена",
			})
		case inlineConfirmMenu:
			err := updateMenu(newMenu)

			if err != nil {
				return err
			}

			c.Edit("Новое меню установлено. Должно прийти ниже:", &tele.ReplyMarkup{})

			delete(listeningForMenu, int(c.Chat().ID))

			chatList, err := getAllChats()

			if err != nil {
				return err
			}

			if debug {
				return c.Send("Добавлено новое меню:\n" + newMenu)
			}

			for _, chatId := range chatList {
				b.Send(tele.ChatID(chatId), "Добавлено новое меню:\n"+newMenu)
			}

			return nil
		case inlineDenyMenu:
			return c.Edit("Меню удалил, слушаю новое меню, если нужно отменить - отправь \"отмена\"")
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

	if lfm, ok := listeningForMenu[chatId]; ok && lfm {
		newMenu = c.Text()

		if len(newMenu) <= 0 {
			return c.Send("Получил пустое сообщение. Всё ещё жду меню")
		}

		if strings.ToLower(newMenu) == "отмена" {
			delete(listeningForMenu, chatId)
			return c.Send("Больше меню не жду")
		}

		keyboardMenu := tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{{
				tele.InlineButton{
					Text: "Всё верно",
					Data: inlineConfirmMenu,
				},
				tele.InlineButton{
					Text: "Нет, всё плохо",
					Data: inlineDenyMenu,
				},
			}},
		}

		return c.Send("Отправляю всем новое меню:\n"+newMenu, &keyboardMenu)
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

				timer = time.AfterFunc(time.Hour, func() {
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

	sub, err := findSubscriptionByChat(int(c.Chat().ID), t)

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

	if sub.ParentId.Valid {
		err = updateSubscriptionStatus(int(sub.ParentId.Int64), Refuse)
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

func createBookingMarkup(subList []Subscription) (*tele.ReplyMarkup, error) {
	var buttonList []tele.Row

	m := &tele.ReplyMarkup{}

	for _, sub := range subList {
		subName := sub.FIO

		if sub.Status == Booked {
			subName += " - забронирован"
		}

		buttonList = append(buttonList, m.Row(m.Data(subName, inlineBook, strconv.Itoa(sub.Id))))
	}

	m.Inline(buttonList...)

	return m, nil
}

func replaceBookingButtons(c tele.Context) error {
	subList, err := findRefusedSubscriptionList(now.BeginningOfDay(), now.EndOfDay(), int(c.Chat().ID))
	if err != nil {
		return err
	}

	if len(subList) <= 0 {
		return c.Edit("❌ Свободных обедов пока нет ❌\n" +
			"Если вы знаете, что кто-то не пришел и не отметил это в боте, пожалуйста, сообщите @DariaRazan. Возможно, обед все же освободится! 🍽😊")
	}

	m, err := createBookingMarkup(subList)

	if err != nil {
		return err
	}

	return c.Edit("🍽 Свободные обеды на сегодня!\n"+
		"Вот список обедов, от которых отказались. Вы можете забронировать один из них и насладиться вкусным обедом без угрызений совести! 😋\n"+
		"Приятного аппетита! 🍴", m)
}
