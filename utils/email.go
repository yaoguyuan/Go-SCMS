package utils

import (
	"auth/initializers"
	"math/rand"
	"regexp"

	"gopkg.in/gomail.v2"
)

// IsValidEmail checks if the provided email address is valid according to a regex pattern.
func IsValidEmail(email string) bool {
	pattern := `^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic("Regex compilation error: " + err.Error())
	}
	return re.MatchString(email)
}

// GenerateVerificationCode generates a random verification code of the specified size.
func GenerateVerificationCode(size int) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, size)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// SendVerificationCode sends a verification code to the user's email address.
func SendVerificationCode(email, code string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", initializers.DAILER.Username)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "VERIFICATION CODE")
	m.SetBody("text/html", "Hello, <br> Your verification code is: <b>"+code+"</b>")

	if err := initializers.DAILER.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
