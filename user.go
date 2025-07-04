package main

import (
	"gopkg.in/gomail.v2"
	"math/rand"
	"os"
	"strconv"
)

var validationCodeList = make(map[int]string)
var chatIdToEmailDict = make(map[int]string)

func SendVerificationCode(email string, chatId int) error {
	validationCodeList[chatId] = strconv.Itoa(100 + rand.Intn(999))
	chatIdToEmailDict[chatId] = email

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_FROM"))
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Подтверждение email")
	m.SetBody("text/plain", "Код подтверждения: "+validationCodeList[chatId])

	return dialer.DialAndSend(m)
}

func clearVerificationCode(chatId int) {
	delete(validationCodeList, chatId)
	delete(chatIdToEmailDict, chatId)
}

func CheckVerificationCode(chatId int, code string) bool {
	return validationCodeList[chatId] == code
}

func GetEmailByChat(chatId int) string {
	return chatIdToEmailDict[chatId]
}

func CodeSent(chatId int) bool {
	return validationCodeList[chatId] != ""
}
