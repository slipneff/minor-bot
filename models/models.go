package models

import (
	"fmt"

	"github.com/google/uuid"
)

type User struct {
	Id           int64 `gorm:"column:id;primaryKey"`
	IsCustomer   int   `gorm:"column:is_customer;primaryKey"`
	Balance      int   `gorm:"column:balance;default:1"`
	CustomerId   int64
	RespondentId int64

	Customer   Customer   `gorm:"foreignKey:CustomerId"`
	Respondent Respondent `gorm:"foreignKey:RespondentId"`
}

type Respondent struct {
	Id         int64 `gorm:"column:id;primaryKey"`
	Name       string
	Age        string
	Gender     string
	Geo        string
	Category   string
	University string
	Job        string
	Rating     int
	Ready      bool
}

func (r *Respondent) ToString() string {
	str := ""
	if r.Name!= "" {
		str += fmt.Sprintf("Name: %s\n", r.Name)
	}
	if r.Age!= "" {
		str += fmt.Sprintf("Age: %s\n", r.Age)
	}
	if r.Gender!= "" {
		str += fmt.Sprintf("Gender: %s\n", r.Gender)
	}
	if r.Geo!= "" {
		str += fmt.Sprintf("Geo: %s\n", r.Geo)
	}
	if r.Category!= "" {
		str += fmt.Sprintf("*Категория*: %s,\n", r.Category)
	}
	if r.University!= "" {
		str += fmt.Sprintf("*Специализация*: %s,\n", r.University)
	}
	if r.Job!= "" {
		str += fmt.Sprintf("*Работа*: %s,\n", r.Job)
	}
	if str == "" {
		str = "*Профиль еще не создан.*"
	}
	return str
}

type Customer struct {
	Id         int64 `gorm:"column:id;primaryKey"`
	Name       string
	Age        string
	Gender     string
	Geo        string
	Category   string
	University string
	Job        string
	Ready      bool
	Desc       string
	Results    string
	Time       string `gorm:"default:'1 час'"`
	Theme      string `gorm:"default:'Без темы'"`
	Count      int
	Available  int `gorm:"default:0"`
}

func (c *Customer) ToString() string {
	str := "*Требования к респонденту*\n"
	str += fmt.Sprintf("\n*Примерный возраст респондента*: %s,", c.Age)
	str += fmt.Sprintf("\n*Пол*: %s,", c.Gender)
	str += fmt.Sprintf("\n*Географическое местоположение*: %s,", c.Geo)
	if c.Category != "" {
		str += fmt.Sprintf("\n*Категория*: %s,", c.Category)
	}
	if c.University != "" {
		str += fmt.Sprintf("\n*Специализация*: %s,", c.University)
	}
	if c.Job!= "" {
		str += fmt.Sprintf("\n*Работа*: %s,", c.Job)
	}
	str += fmt.Sprintf("\n*Тема*: %s,", c.Theme)
	str += fmt.Sprintf("\n*Длительность*: %s,", c.Time)
	str += fmt.Sprintf("\n*Доступно*: %s,", fmt.Sprintf("%d/%d", c.Count-c.Available, c.Count))
	str += fmt.Sprintf("\n*Интервьюер*: %s,\n*Готов поделиться результатом?*: %s", c.Name, c.Results)
	str += fmt.Sprintf("\n*Комментарий*: %s,", c.Desc)
	return str
}
func (c *Customer) ShortString() string {
	return fmt.Sprintf("*Тема*: %s,\n*Длительность*: %s,\n*Доступно*: %s,\n*Интервьюер*: %s,\n*Готов поделиться результатом?*: %s,\n*Комментарий*: %s",
		c.Theme, c.Time, fmt.Sprintf("%d/%d", c.Count-c.Available, c.Count), c.Name, c.Results, c.Desc)

}

type Interview struct {
	CustomerId           int64     `gorm:"column:customer_id;primaryKey"`
	ApplicationId        uuid.UUID `gorm:"column:application_id;primaryKey"`
	RespondentId         int64     `gorm:"column:respondent_id;primaryKey"`
	RespondentName       string    `gorm:"column:respondent_name"`
	ApprovedByCustomer   bool      `gorm:"column:approved_cust;default:false"`
	ApprovedByRespondent bool      `gorm:"column:approved_resp;default:false"`
	Active               bool      `gorm:"column:active;default:true"`
}
