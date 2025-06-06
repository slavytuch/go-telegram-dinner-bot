package main

import (
	"fmt"
	"math/rand"
	"strconv"
)

var validationCodeList = make(map[int]string)
var chatIdToEmailDict = make(map[int]string)

func SendVerificationCode(email string, chatId int) error {
	validationCodeList[chatId] = strconv.Itoa(100 + rand.Intn(999))
	chatIdToEmailDict[chatId] = email

	//TODO: send email
	fmt.Println("Verification code sent:" + validationCodeList[chatId])

	return nil
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
