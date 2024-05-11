package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/slipneff/minor-bot/config"
	"github.com/slipneff/minor-bot/constants"
	"github.com/slipneff/minor-bot/models"
	sql "github.com/slipneff/minor-bot/storage"
	"gorm.io/gorm"
)

type Scene int

const (
	SceneHello Scene = iota
	SceneAskSide
	SceneCreateCustomer
	SceneAskName
	SceneAskAge
	SceneAskGender
	SceneAskGeo
	SceneAskDifferentGeo
	SceneAskCategory
	SceneAskUniversity
	SceneAskJob
	SceneAskDifferentUniversity
	SceneAskDifferentJob
	SceneAskIsReady
	SceneApproveInterviewRespondent
	SceneApproveInterviewCustomer
	SceneRateRespondent
)

type UserSession struct {
	CurrentScene Scene
	User         models.User
}

var sessions map[int64]*UserSession

func main() {
	ctx := context.Background()
	config := config.MustLoadConfig("config.yaml")
	sql := sql.New(sql.MustNewSQLite(config), trmgorm.DefaultCtxGetter)
	bot, err := tgbotapi.NewBotAPI("6547512514:AAHJpIeMLJZMHAnC608UKe16HQlLDjSqJDY")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		log.Panic(err)
	}
	sessions = make(map[int64]*UserSession)

	for update := range updates {

		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		userID := update.Message.Chat.ID

		if sessions[userID] == nil {
			sessions[userID] = &UserSession{
				CurrentScene: SceneHello,
			}
		}

		session := sessions[userID]
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(userID, constants.Hello)
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Я хочу проводить интервью"),
						tgbotapi.NewKeyboardButton("Я хочу отвечать на интервью"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskSide
			case "help":
				msg := tgbotapi.NewMessage(userID, constants.Hello)
				bot.Send(msg)
			case "ready_to_answer":
				err := sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:    userID,
					Ready: true,
				})
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						msg := tgbotapi.NewMessage(userID, "Профиль не найден")
						bot.Send(msg)
					} else {
						log.Println(err)
						msg := tgbotapi.NewMessage(userID, "Произошла ошибка при обновлении профиля. Попробуйте позже.")
						bot.Send(msg)
						continue
					}
				}
				msg := tgbotapi.NewMessage(userID, "Ожидайте новых заказов!")
				bot.Send(msg)
				continue
			case "ready_to_ask":
				err := sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:    userID,
					Ready: true,
				})
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						msg := tgbotapi.NewMessage(userID, "Профиль не найден")
						bot.Send(msg)
					} else {
						log.Println(err)
						msg := tgbotapi.NewMessage(userID, "Произошла ошибка при обновлении профиля. Попробуйте позже.")
						bot.Send(msg)
						continue
					}
				}
				msg := tgbotapi.NewMessage(userID, "Ожидайте новых заказов!")
				bot.Send(msg)
				continue
			case "my_interview":
				interviews, err := sql.FindInterviewByCustomerId(ctx, userID)
				if err != nil {
					log.Println(err)
					msg := tgbotapi.NewMessage(userID, "Произошла ошибка при получении интервью. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Ваши активные интервью:")
				for _, interview := range interviews {
					msg.Text += fmt.Sprintf("\n%d", interview.RespondentId)
				}
				bot.Send(msg)
			case "rate":
				interviews, err := sql.FindInterviewByCustomerId(ctx, userID)
				if err != nil {
					log.Println(err)
					msg := tgbotapi.NewMessage(userID, "Произошла ошибка при получении интервью. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Кого вы хотите оценить? Напишите в формате {ID} {Оценка 1-5}")
				for _, interview := range interviews {
					msg.Text += fmt.Sprintf("\n%d", interview.RespondentId)
				}
				bot.Send(msg)
				session.CurrentScene = SceneRateRespondent
			default:
				msg := tgbotapi.NewMessage(userID, "Неизвестная команда")
				bot.Send(msg)
				continue
			}
		}
		switch session.CurrentScene {
		case SceneHello:
			msg := tgbotapi.NewMessage(userID, constants.Hello)
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Я хочу проводить интервью"),
					tgbotapi.NewKeyboardButton("Я хочу отвечать на интервью"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskSide
		case SceneAskSide:
			switch update.Message.Text {
			case "Я хочу отвечать на интервью":
				err := sql.CreateUser(ctx, &models.User{
					Id:         userID,
					IsCustomer: 0,
				})
				if err != nil {
					if err == gorm.ErrDuplicatedKey {
						msg := tgbotapi.NewMessage(userID, "Профиль уже создан, переходим к следующему шагу")
						bot.Send(msg)
					} else {
						log.Println(err)
						msg := tgbotapi.NewMessage(userID, "Произошла ошибка при создании профиля. Попробуйте позже.")
						bot.Send(msg)
						continue
					}
				}
				err = sql.CreateRespondent(ctx, &models.Respondent{
					Id: userID,
				})
				if err != nil {
					if err != gorm.ErrDuplicatedKey {
						log.Println(err)
						msg := tgbotapi.NewMessage(userID, "Произошла ошибка при создании профиля. Попробуйте позже.")
						bot.Send(msg)
						continue
					}
				}
				session.CurrentScene = SceneAskName
				msg := tgbotapi.NewMessage(userID, "Отлично, а теперь необходимо ввести некоторые данные:\nУкажите, как к вам обращаться?")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				bot.Send(msg)

			case "Я хочу проводить интервью":
				err := sql.CreateUser(ctx, &models.User{
					Id:         userID,
					IsCustomer: 1,
				})
				if err != nil {
					if err == gorm.ErrDuplicatedKey {
						msg := tgbotapi.NewMessage(userID, "Профиль уже создан, переходим к следующему шагу")
						bot.Send(msg)
					} else {
						log.Println(err)
						msg := tgbotapi.NewMessage(userID, "Произошла ошибка при создании профиля. Попробуйте позже.")
						bot.Send(msg)
						continue
					}
				}
				session.CurrentScene = SceneCreateCustomer
				msg := tgbotapi.NewMessage(userID, "Отлично, а теперь необходимо ввести некоторые данные:\nУкажите количество человек, у которых вы хотите провести интервью")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				bot.Send(msg)
			default:
				msg := tgbotapi.NewMessage(userID, "Выберите, что вы хотите сделать")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Я хочу проводить интервью"),
						tgbotapi.NewKeyboardButton("Я хочу отвечать на интервью"),
					),
				)
				bot.Send(msg)
				continue
			}
			if update.Message.Text == "Я хочу проводить интервью" {
				session.User.IsCustomer = 1
			} else {
				session.User.IsCustomer = 0
			}

		case SceneCreateCustomer:
			count, err := strconv.Atoi(update.Message.Text)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Введите корректное количество человек"))
				continue
			}
			err = sql.CreateCustomer(ctx, &models.Customer{
				UserId: userID,
			})
			if err != nil && err != gorm.ErrDuplicatedKey {
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при создании заявки, попробуйте еще раз"))
				continue
			}
			err = sql.UpdateCustomerByUserId(ctx, models.Customer{
				UserId: userID,
				Count:  count,
			})
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при создании заявки, попробуйте еще раз"))
				continue
			}
			msg := tgbotapi.NewMessage(userID, "Отлично, а теперь необходимо ввести некоторые данные:\nУкажите, как к вам обращаться?")
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)
			session.CurrentScene = SceneAskName
		case SceneAskName:
			var message string
			if session.User.IsCustomer == 1 {
				message = "Введите возраст респондента"
				session.User.Customer.Name = update.Message.Text
			} else {
				message = "Отлично, теперь необходимо ввести ваш возраст"
				session.User.Respondent.Name = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, message)
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)
			session.CurrentScene = SceneAskAge
		case SceneAskAge:
			age, err := strconv.Atoi(update.Message.Text)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Введите корректный возраст"))
				continue
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Age:    int32(age),
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Age = int32(age)
			}
			msg := tgbotapi.NewMessage(userID, "Отлично, теперь выберете пол")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Мужской"),
					tgbotapi.NewKeyboardButton("Женский"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskGender
		case SceneAskGender:
			if update.Message.Text != "Мужской" && update.Message.Text != "Женский" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный пол")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Мужской"),
						tgbotapi.NewKeyboardButton("Женский"),
					),
				)
				bot.Send(msg)
				continue
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Gender: update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Gender = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Выберите географическое местоположение")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Екатеринбург"),
					tgbotapi.NewKeyboardButton("Санкт-Петербург"),
					tgbotapi.NewKeyboardButton("Москва и МО"),
				),tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Казань"),
					tgbotapi.NewKeyboardButton("Город неважен"),
					tgbotapi.NewKeyboardButton("Другое"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskGeo
		case SceneAskGeo:
			if update.Message.Text != "Екатеринбург" && update.Message.Text != "Санкт-Петербург" && update.Message.Text != "Москва и МО" && update.Message.Text != "Казань" && update.Message.Text != "Город неважен" && update.Message.Text != "Другое" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректное местоположение")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Екатеринбург"),
						tgbotapi.NewKeyboardButton("Санкт-Петербург"),
						tgbotapi.NewKeyboardButton("Москва и МО"),
					),tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Казань"),
						tgbotapi.NewKeyboardButton("Город неважен"),
						tgbotapi.NewKeyboardButton("Другое"),
					),
				)
				bot.Send(msg)
				continue
			}
			if update.Message.Text == "Другое" {
				msg := tgbotapi.NewMessage(userID, "Введите название города")
				bot.Send(msg)
				session.CurrentScene = SceneAskDifferentGeo
				continue
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Geo:    update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Geo = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Выберите категорию")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Студент"),
					tgbotapi.NewKeyboardButton("Предприниматель"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Работник компании"),
					tgbotapi.NewKeyboardButton("Все категории"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskCategory
		case SceneAskDifferentGeo:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Geo:    update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Geo = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Выберите категорию")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Студент"),
					tgbotapi.NewKeyboardButton("Предприниматель"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Работник компании"),
					tgbotapi.NewKeyboardButton("Все категории"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskCategory
		case SceneAskCategory:
			if update.Message.Text != "Студент" && update.Message.Text != "Предприниматель" && update.Message.Text != "Работник компании" && update.Message.Text != "Все категории" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректную категорию")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Студент"),
						tgbotapi.NewKeyboardButton("Предприниматель"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Работник компании"),
						tgbotapi.NewKeyboardButton("Все категории"),
					),
				)
				bot.Send(msg)
				continue
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId:   userID,
					Category: update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Category = update.Message.Text
			}
			if update.Message.Text == "Студент" {
				msg := tgbotapi.NewMessage(userID, "Факультет респондента?")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Экономика"),
						tgbotapi.NewKeyboardButton("Психология"),
						tgbotapi.NewKeyboardButton("Маркетинг"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Юриспруденция"),
						tgbotapi.NewKeyboardButton("Другое"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskUniversity
				continue
			}
			if update.Message.Text == "Работник компании" {
				msg := tgbotapi.NewMessage(userID, "Cфера респондента?")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Бизнес"),
						tgbotapi.NewKeyboardButton("Медицина"),
						tgbotapi.NewKeyboardButton("IT"),
						
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Строительство"),
						tgbotapi.NewKeyboardButton("Другое"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskJob
				continue
			}
			msg := tgbotapi.NewMessage(userID, "Готовы сделать интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Готов к интервью"),
					tgbotapi.NewKeyboardButton("Не готов"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskUniversity:
			if update.Message.Text != "Экономика" && update.Message.Text != "Психология" && update.Message.Text != "Маркетинг" && update.Message.Text != "Юриспруденция" && update.Message.Text != "Другое" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный факультет")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Экономика"),
						tgbotapi.NewKeyboardButton("Психология"),
						tgbotapi.NewKeyboardButton("Маркетинг"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Юриспруденция"),
						tgbotapi.NewKeyboardButton("Другое"),
					),
				)
				bot.Send(msg)
				continue
			}
			if update.Message.Text == "Другое" {
				msg := tgbotapi.NewMessage(userID, "Введите название факультета")
				bot.Send(msg)
				session.CurrentScene = SceneAskDifferentUniversity
				continue
			}

			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId:     userID,
					University: update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.University = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Готовы сделать интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Готов к интервью"),
					tgbotapi.NewKeyboardButton("Не готов"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskDifferentUniversity:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId:     userID,
					University: update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.University = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Готовы сделать интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Готов к интервью"),
					tgbotapi.NewKeyboardButton("Не готов"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskJob:
			if update.Message.Text != "Бизнес" && update.Message.Text != "Медицина" && update.Message.Text != "Строительство" && update.Message.Text != "IT" && update.Message.Text != "Другое" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректную сферу")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Бизнес"),
						tgbotapi.NewKeyboardButton("Медицина"),
						tgbotapi.NewKeyboardButton("IT"),
						
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Строительство"),
						tgbotapi.NewKeyboardButton("Другое"),
					),
				)
				bot.Send(msg)
				continue
			}
			if update.Message.Text == "Другое" {
				msg := tgbotapi.NewMessage(userID, "Введите название сферы")
				bot.Send(msg)
				session.CurrentScene = SceneAskDifferentJob
				continue
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Job:    update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Job = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Готовы сделать интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Готов к интервью"),
					tgbotapi.NewKeyboardButton("Не готов"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskDifferentJob:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					UserId: userID,
					Job:    update.Message.Text,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Job = update.Message.Text
			}
			msg := tgbotapi.NewMessage(userID, "Готовы сделать интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Готов к интервью"),
					tgbotapi.NewKeyboardButton("Не готов"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskIsReady:
			if update.Message.Text != "Готов к интервью" && update.Message.Text != "Не готов" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный ответ")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Готов к интервью"),
						tgbotapi.NewKeyboardButton("Не готов"),
					),
				)
				bot.Send(msg)
				continue
			}
			if session.User.IsCustomer == 1 {
				if update.Message.Text == "Готов к интервью" {
					session.User.Customer.Ready = true
					err := sql.UpdateCustomerByUserId(ctx, session.User.Customer)
					if err != nil {
						bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
						continue
					}
				}
			} else {
				if update.Message.Text == "Готов к интервью" {
					session.User.Respondent.Ready = true
					err := sql.UpdateRespondentByUserId(ctx, session.User.Respondent)
					if err != nil {
						bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
						continue
					}
				}
			}

			if update.Message.Text == "Готов к интервью" {
				msg := tgbotapi.NewMessage(userID, "Ожидайте новых заказов!")
				bot.Send(msg)
				continue
			} else {
				msg := tgbotapi.NewMessage(userID, "Как будете готовы, напишите мне /ready")
				bot.Send(msg)
				continue
			}
		case SceneApproveInterviewCustomer:
			if update.Message.Text != "Согласен" && update.Message.Text != "Отказать" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный ответ")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Согласен"),
						tgbotapi.NewKeyboardButton("Отказать"),
					),
				)
				bot.Send(msg)
				continue
			}
			interview, err := sql.GetLastInterviewByCustomer(ctx, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при поиске интервью, попробуйте еще раз"))
				continue
			}
			if update.Message.Text == "Согласен" {
				err := sql.ApproveInterviewByCustomer(ctx, &models.Interview{
					CustomerId:           interview.CustomerId,
					ApplicationId:        interview.ApplicationId,
					RespondentId:         interview.RespondentId,
					ApprovedByCustomer:   true,
					ApprovedByRespondent: interview.ApprovedByRespondent,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении статуса, попробуйте еще раз"))
					continue
				}
				if interview.ApprovedByRespondent {
					msg := tgbotapi.NewMessage(interview.CustomerId, fmt.Sprintf("Респондент так же согласен. Его айди: %d", interview.RespondentId))
					bot.Send(msg)
				}
				msg := tgbotapi.NewMessage(userID, "Спасибо, интервьюер с вами свяжется для назначения времени и даты интервью.")
				bot.Send(msg)
				continue
			} else {
				err := sql.DeleteInterviewByRespondentID(ctx, interview.RespondentId)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при удалении интервью, попробуйте еще раз"))
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Хорошо, мы отменили запрос на интервью")
				bot.Send(msg)

				continue
			}

		case SceneApproveInterviewRespondent:
			if update.Message.Text != "Согласен" && update.Message.Text != "Отказать" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректную категорию")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Согласен"),
						tgbotapi.NewKeyboardButton("Отказать"),
					),
				)
				bot.Send(msg)
				continue
			}
			interview, err := sql.FindInterviewByRespondentId(ctx, session.User.Respondent.Id)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при поиске интервью, попробуйте еще раз"))
				continue
			}
			if update.Message.Text == "Согласен" {
				err := sql.ApproveInterviewByCustomer(ctx, &models.Interview{
					CustomerId:           interview.CustomerId,
					ApplicationId:        interview.ApplicationId,
					RespondentId:         interview.RespondentId,
					ApprovedByCustomer:   interview.ApprovedByCustomer,
					ApprovedByRespondent: true,
				})
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении статуса, попробуйте еще раз"))
					continue
				}
				if interview.ApprovedByCustomer {
					msg := tgbotapi.NewMessage(interview.CustomerId, fmt.Sprintf("Респондент так же согласен. Его айди: %d", interview.RespondentId))
					bot.Send(msg)
				}
				msg := tgbotapi.NewMessage(userID, "Спасибо, интервьюер с вами свяжется для назначения времени и даты интервью.")
				bot.Send(msg)
				continue
			} else {
				err := sql.DeleteInterviewByRespondentID(ctx, userID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при удалении интервью, попробуйте еще раз"))
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Хорошо, мы отменили запрос на интервью")
				bot.Send(msg)

				continue
			}
		case SceneRateRespondent:
			m := strings.Split(update.Message.Text, " ")
			if len(m) < 2 {
				if err != nil {
					bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
					continue
				}
			}
			id, err := strconv.Atoi(m[0])
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}
			rate, err := strconv.Atoi(m[1])
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Неверная оценка, попробуйте еще раз"))
				continue
			}
			if rate > 5 || rate < 1 {
				bot.Send(tgbotapi.NewMessage(userID, "Неверная оценка, попробуйте еще раз"))
				continue
			}
			resp, err := sql.GetRespondentById(ctx, int64(id))
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}
			err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
				Id:     userID,
				Rating: (resp.Rating + rate) / 2,
			})
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}
			err = sql.PlusOneBalanceUser(ctx, resp.Id)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}
			err = sql.MinusOneBalanceUser(ctx, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}
			err = sql.DeleteInterviewByRespondentID(ctx, resp.Id)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "Респондент не найден, попробуйте еще раз"))
				continue
			}

		}

	}
}
