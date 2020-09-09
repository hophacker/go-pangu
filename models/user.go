package models

import (
	"errors"
	"fmt"
	"go-pangu/db"

	"gorm.io/gorm"
)

type User struct {
	Model
	Email             string `gorm:"index:idx_email,unique"`
	EncryptedPassword string
}

func FindUserByEmail(email string) *User {
	var user User
	db.DB.Where("email = ?", email).First(&user)
	return &user
}

func FindUserById(id string) *User {
	var user User
	db.DB.Where("id = ?", id).First(&user)
	return &user
}

func FindUserByColum(colum string, value interface{}) (*User, bool) {
	var user User
	qs := fmt.Sprintf("%s = ?", colum)
	err := db.DB.Where(qs, value).First(&user).Error
	return &user, errors.Is(err, gorm.ErrRecordNotFound)
}
