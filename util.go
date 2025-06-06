package main

import (
	"math/rand"
	"slices"
	"time"
)

var defaultGreets = []string{
	"Приветствую",
	"Здравствуй",
	"Привет",
	"Рад видеть тебя",
	"Здаров",
}

var morningGreets = []string{
	"Доброе утро",
	"Ты сегодня рановато",
	"Ни свет, ни заря",
}

var eveningGreets = []string{
	"Добрый вечер",
}

var nightGreets = []string{
	"Доброй ночи",
	"Обычно в это время люди спят",
	"Ложись спать",
}

func randomGreet() string {
	var greets = defaultGreets

	hour, _, _ := time.Now().Clock()

	if hour > 6 && hour < 11 {
		greets = slices.Concat(greets, morningGreets)
	}

	if hour > 17 && hour < 21 {
		greets = slices.Concat(greets, eveningGreets)
	}

	if hour > 21 || hour < 6 {
		greets = slices.Concat(greets, nightGreets)
	}

	return greets[rand.Intn(len(greets))]
}
