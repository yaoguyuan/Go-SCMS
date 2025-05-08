package initializers

import (
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

var DAILER *gomail.Dialer

func InitEmail() {
	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		panic("Failed to convert SMTP_PORT to int: " + err.Error())
	}
	DAILER = gomail.NewDialer(os.Getenv("SMTP_HOST"), smtpPort, os.Getenv("EMAIL_ADDR"), os.Getenv("EMAIL_PASS"))
}
