package models

import "fmt"

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
	return fmt.Sprintf(" Имя: %s,\n Возраст: %s,\n Пол: %s,\n Местоположение: %s,\n Категория: %s,\n Университет: %s,\n Работа: %s,\n Готов к выполнению: %t",
		r.Name, r.Age, r.Gender, r.Geo, r.Category, r.University, r.Job, r.Ready)
}

type Customer struct {
	Id         int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserId     int64
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
	Count      int
}

func (c *Customer) ToString() string {
	return fmt.Sprintf("*Имя заявителя*: %s,\n*Средний возраст респондентов*: %s,\n*Количество респондентов*: %d,\n*Пол респондетов*: %s,\n*География респондента*: %s,\n*Категория респондента*: %s",
		c.Name, c.Age, c.Count,c.Gender, c.Geo, c.Category)
}

type Interview struct {
	CustomerId           int64 `gorm:"column:customer_id;primaryKey"`
	ApplicationId        int64 `gorm:"column:application_id;primaryKey"`
	RespondentId         int64 `gorm:"column:respondent_id;primaryKey"`
	ApprovedByCustomer   bool  `gorm:"column:approved_cust;default:false"`
	ApprovedByRespondent bool  `gorm:"column:approved_resp;default:false"`
	Active               bool  `gorm:"column:active;default:true"`
}
