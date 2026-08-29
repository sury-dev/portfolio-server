package utils

import "golang.org/x/crypto/bcrypt"

func HashString(str string) (string, error) {
	hashedStr, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedStr), nil
}

func CompareString(hashedStr, str string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(str))
}