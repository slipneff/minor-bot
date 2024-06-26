package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
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
	SceneAskDescription
	SceneAskUniversity
	SceneAskJob
	SceneAskDifferentUniversity
	SceneAskDifferentJob
	SceneAskIsReady
	SceneApproveInterviewRespondent
	SceneApproveInterviewCustomer
	SceneRateRespondent
	SceneAskTime
	SceneMenu
	SceneAskResults
	SceneAskComment
	SceneAskTheme
	SceneAskIsJob
	SceneHelp
)

type UserSession struct {
	CurrentScene Scene
	User         models.User
}
type customerForDashboard struct {
	Id       int64
	Customer *models.Customer
}

var sessions map[int64]*UserSession

func getNextCustomerForDashboard(ctx context.Context, sql *sql.Storage, userID int64) (*customerForDashboard, error) {
	res := new(customerForDashboard)
	resp, err := sql.GetNextReadyCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(resp) > 1 {
		res.Id = resp[1].Id
		res.Customer = resp[0]
	} else if len(resp) > 0 {
		res.Customer = resp[0]
	} else {
		return nil, fmt.Errorf("no customers found")
	}
	return res, nil
}
func getPrevCustomerForDashboard(ctx context.Context, sql *sql.Storage, userID int64) (*customerForDashboard, error) {
	res := new(customerForDashboard)
	resp, err := sql.GetPrevReadyCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(resp) > 1 {
		res.Id = resp[1].Id
		res.Customer = resp[0]
	} else if len(resp) > 0 {
		res.Customer = resp[0]
	} else {
		return nil, fmt.Errorf("no customers found")
	}
	return res, nil
}
func responseRespondent(ctx context.Context, bot *tgbotapi.BotAPI, sql *sql.Storage, customer_id, respondent_id int64, username string) error {
	err := sql.CreateInterview(ctx, &models.Interview{
		CustomerId:           customer_id,
		ApplicationId:        uuid.New(),
		RespondentId:         respondent_id,
		RespondentName:       username,
		ApprovedByCustomer:   false,
		ApprovedByRespondent: true,
		Active:               true,
	})
	if err != nil {
		return err
	}
	resp, err := sql.GetRespondentById(ctx, respondent_id)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(customer_id, "@"+username+" откликнулся на ваш заказ:\n\n"+resp.ToString())
	msg.ParseMode = "markdown"
	bot.Send(msg)
	msg = tgbotapi.NewMessage(respondent_id, "Ваш отклик отправлен")
	bot.Send(msg)

	return nil
}
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

	updates := bot.GetUpdatesChan(u)
	cmdCfg := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{
			Command:     "restart",
			Description: "Сброс",
		},
		tgbotapi.BotCommand{
			Command:     "start",
			Description: "Старт",
		},
		tgbotapi.BotCommand{
			Command:     "menu",
			Description: "Главное меню",
		},
		tgbotapi.BotCommand{
			Command:     "help",
			Description: "Помощь",
		},
	)
	bot.Send(cmdCfg)
	sessions = make(map[int64]*UserSession)

	for update := range updates {
		if update.CallbackQuery != nil {
			c := tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)
			_, err := bot.Send(c)
			if err != nil {
				log.Println(err)
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "next_customer_") {
				id, err := strconv.ParseInt(strings.Split(update.CallbackQuery.Data, "customer_")[1], 10, 64)
				if err != nil {
					log.Println(err)
					continue
				}
				user, err := getNextCustomerForDashboard(ctx, sql, id)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Доска заказов:\n\n%s", user.Customer.ShortString()))
				msg.ParseMode = "markdown"
				if user.Id != 0 {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", fmt.Sprintf("prev_customer_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", fmt.Sprintf("next_customer_%d", user.Customer.Id)),
						),
					)
				} else {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", fmt.Sprintf("prev_customer_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", update.CallbackQuery.Data),
						),
					)
				}
				bot.Send(msg)
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "prev_customer_") {
				id, err := strconv.ParseInt(strings.Split(update.CallbackQuery.Data, "customer_")[1], 10, 64)
				if err != nil {
					log.Println(err)
					continue
				}
				user, err := getPrevCustomerForDashboard(ctx, sql, id)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Доска заказов:\n\n%s", user.Customer.ShortString()))
				msg.ParseMode = "markdown"
				if user.Id != 0 {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", update.CallbackQuery.Data),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", update.CallbackQuery.Data),
						),
					)
				} else {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", update.CallbackQuery.Data),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", fmt.Sprintf("next_customer_%d", user.Id)),
						),
					)
				}
				bot.Send(msg)
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "full_") {
				id, err := strconv.ParseInt(strings.Split(update.CallbackQuery.Data, "_")[1], 10, 64)
				if err != nil {
					log.Println(err)
					continue
				}
				user, err := sql.GetCustomerByUserId(ctx, id)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Доска заказов:\n\n%s", user.ToString()))
				msg.ParseMode = "markdown"
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						//id_customer | id_respondent
						tgbotapi.NewInlineKeyboardButtonData("Откликнуться", fmt.Sprintf("respond_%d_%d", user.Id, update.CallbackQuery.Message.Chat.ID)),
						tgbotapi.NewInlineKeyboardButtonData("Назад", "dashboard"),
					),
				)
				bot.Send(msg)
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "respond_") {
				customer_id, err := strconv.ParseInt(strings.Split(update.CallbackQuery.Data, "_")[1], 10, 64)
				if err != nil {
					log.Println(err)
					continue
				}
				respondent_id, err := strconv.ParseInt(strings.Split(update.CallbackQuery.Data, "_")[2], 10, 64)
				if err != nil {
					log.Println(err)
					continue
				}
				err = responseRespondent(ctx, bot, sql, customer_id, respondent_id, update.CallbackQuery.Message.Chat.UserName)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при отклике. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
			}
			switch update.CallbackQuery.Data {
			case "balance":
				balance, err := sql.GetBalanceUser(ctx, update.CallbackQuery.Message.Chat.ID)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении баланса. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Ваш баланс: %d", balance))
				bot.Send(msg)
			case "profile_customer":
				s, err := sql.GetCustomerByUserId(ctx, update.CallbackQuery.Message.Chat.ID)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%s", s.ToString()))
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case "profile_respondent":
				s, err := sql.GetRespondentById(ctx, update.CallbackQuery.Message.Chat.ID)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Ваш профиль:\n\n%s", s.ToString()))
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case "dashboard":
				user, err := getNextCustomerForDashboard(ctx, sql, 0)
				if err != nil {
					fmt.Println(err)
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Доска заказов:\n\n%s", user.Customer.ShortString()))
				msg.ParseMode = "markdown"
				if user.Id != 0 {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", update.CallbackQuery.Data),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", fmt.Sprintf("next_customer_%d", user.Customer.Id)),
						),
					)
				} else {
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("<--", update.CallbackQuery.Data),
							tgbotapi.NewInlineKeyboardButtonData("Подробнее", fmt.Sprintf("full_%d", user.Customer.Id)),
							tgbotapi.NewInlineKeyboardButtonData("-->", update.CallbackQuery.Data),
						),
					)
				}
				bot.Send(msg)

			case "my_respondents":
				respondents, err := sql.GetRespondentsByCustomerId(ctx, update.CallbackQuery.Message.Chat.ID)
				if err != nil {
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Произошла ошибка при получении профиля. Попробуйте позже.")
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ваши респонденты:\n")
				for i, respondent := range respondents {
					msg.Text += fmt.Sprintf("*%d.* @%s\n", i+1, respondent.RespondentName)
				}
				msg.ParseMode = "markdown"
				bot.Send(msg)
			}
		}
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
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Я хочу проводить интервью"),
						tgbotapi.NewKeyboardButton("Я хочу отвечать на интервью"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskSide
			case "help":
				msg := tgbotapi.NewMessage(userID, constants.Help)
				bot.Send(msg)
				session.CurrentScene = SceneHelp
			case "restart":
				sql.ResetAll(ctx, userID)
				msg := tgbotapi.NewMessage(userID, "---Настройки сброшены---\n"+constants.Hello)
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Я хочу проводить интервью"),
						tgbotapi.NewKeyboardButton("Я хочу отвечать на интервью"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskSide
			case "menu":
				session.CurrentScene = SceneMenu
				msg := tgbotapi.NewMessage(userID, "Меню")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Мои респонденты", "my_respondents"),
						tgbotapi.NewInlineKeyboardButtonData("Ваши заявки", "profile_customer"),
						tgbotapi.NewInlineKeyboardButtonData("Профиль респондента", "profile_respondent"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Баланс", "balance"),
						tgbotapi.NewInlineKeyboardButtonData("Доска исследователя", "dashboard"),
					),
				)
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
				msg := tgbotapi.NewMessage(userID, "Отлично, теперь вы можете откликаться на заявки на доске исследователя.")
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
				msg := tgbotapi.NewMessage(userID, "Ваше объявление выставлено на доску исследователя.\nВы можете редактировать свою заявку и просматривать откликнувшихся в личном кабинете и меню")
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
			balance, err := sql.GetBalanceUser(ctx, userID)
			if err != nil {
				log.Println(err)
				msg := tgbotapi.NewMessage(userID, "Произошла ошибка при получении баланса. Попробуйте позже.")
				bot.Send(msg)
				continue
			}
			if balance+1 < count {
				msg := tgbotapi.NewMessage(userID, fmt.Sprintf("У вас недостаточно средств. Ваш баланс: %d\nВведите корректное число от 1 до %d", balance, balance))
				bot.Send(msg)
				continue
			}
			err = sql.CreateCustomer(ctx, &models.Customer{
				Id: userID,
			})
			if err != nil && err != gorm.ErrDuplicatedKey {
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при создании заявки, попробуйте еще раз"))
				continue
			}
			err = sql.UpdateCustomerByUserId(ctx, models.Customer{
				Id:    userID,
				Count: count,
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
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:   userID,
					Name: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				message = "Отлично, теперь необходимо ввести ваш возраст"
				session.User.Respondent.Name = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:   userID,
					Name: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			msg := tgbotapi.NewMessage(userID, message)
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Неважно"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskAge
		case SceneAskAge:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:  userID,
					Age: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Отлично, теперь выберете пол респондента")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Мужской"),
						tgbotapi.NewKeyboardButton("Женский"),
						tgbotapi.NewKeyboardButton("Неважно"),
					),
				)
				bot.Send(msg)
			} else {
				session.User.Respondent.Age = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:  userID,
					Age: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				msg := tgbotapi.NewMessage(userID, "Отлично, теперь выберете ваш пол")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Мужской"),
						tgbotapi.NewKeyboardButton("Женский"),
					),
				)
				bot.Send(msg)
			}

			session.CurrentScene = SceneAskGender
		case SceneAskGender:
			if update.Message.Text != "Мужской" && update.Message.Text != "Женский" && update.Message.Text != "Неважно" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный пол")
				if session.User.IsCustomer == 1 {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Мужской"),
							tgbotapi.NewKeyboardButton("Женский"),
							tgbotapi.NewKeyboardButton("Неважно"),
						),
					)
				} else {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Мужской"),
							tgbotapi.NewKeyboardButton("Женский"),
						),
					)
					bot.Send(msg)
					continue
				}
			}
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:     userID,
					Gender: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Gender = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:     userID,
					Gender: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			msg := tgbotapi.NewMessage(userID, "Выберите географическое местоположение")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Екатеринбург"),
					tgbotapi.NewKeyboardButton("Санкт-Петербург"),
					tgbotapi.NewKeyboardButton("Москва и МО"),
				), tgbotapi.NewKeyboardButtonRow(
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
					), tgbotapi.NewKeyboardButtonRow(
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
					Id:  userID,
					Geo: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Geo = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:  userID,
					Geo: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
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
					Id:  userID,
					Geo: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Geo = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:  userID,
					Geo: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
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
					Id:       userID,
					Category: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Category = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:       userID,
					Category: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			if update.Message.Text == "Студент" {
				msg := tgbotapi.NewMessage(userID, "Факультет респондента?")
				if session.User.IsCustomer == 1 {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Экономика"),
							tgbotapi.NewKeyboardButton("Психология"),
							tgbotapi.NewKeyboardButton("Маркетинг"),
						),
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Юриспруденция"),
							tgbotapi.NewKeyboardButton("Другое"),
							tgbotapi.NewKeyboardButton("Неважно"),
						),
					)
				} else {
					if session.User.IsCustomer == 0 {
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
					}
				}
				bot.Send(msg)
				session.CurrentScene = SceneAskUniversity
				continue
			}
			if update.Message.Text == "Работник компании" {
				msg := tgbotapi.NewMessage(userID, "Cфера респондента?")
				if session.User.IsCustomer == 1 {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Бизнес"),
							tgbotapi.NewKeyboardButton("Медицина"),
							tgbotapi.NewKeyboardButton("IT"),
						),
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Строительство"),
							tgbotapi.NewKeyboardButton("Другое"),
							tgbotapi.NewKeyboardButton("Неважно"),
						),
					)
				} else {
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
				}

				bot.Send(msg)
				session.CurrentScene = SceneAskJob
				continue
			}
			if session.User.IsCustomer == 1 {
				msg := tgbotapi.NewMessage(userID, "Укажите продолжительность интервью")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				bot.Send(msg)
				session.CurrentScene = SceneAskTime
				continue
			}

			msg := tgbotapi.NewMessage(userID, session.User.Customer.ToString()+"\n Готовы пройти интервью?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskTime:
			if update.Message.Text == "" {
				msg := tgbotapi.NewMessage(userID, "Введите продолжительность интервью")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				bot.Send(msg)
				continue
			}
			err = sql.UpdateCustomerByUserId(ctx, models.Customer{
				Id:   userID,
				Time: update.Message.Text,
			})
			if err != nil {
				log.Fatal(err)
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
				continue
			}
			session.CurrentScene = SceneAskTheme
			msg := tgbotapi.NewMessage(userID, "Тема интервью?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Пропустить"),
				),
			)
			bot.Send(msg)
		case SceneAskTheme:
			if update.Message.Text == "" {
				msg := tgbotapi.NewMessage(userID, "Тема интервью?")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Пропустить"),
					),
				)
				bot.Send(msg)
			}
			if update.Message.Text != "Пропустить" {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:    userID,
					Theme: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}

			session.CurrentScene = SceneAskResults
			msg := tgbotapi.NewMessage(userID, "Готовы ли вы поделиться результатами своего исследования?")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Нет"),
				),
			)
			bot.Send(msg)

		case SceneAskResults:
			if update.Message.Text == "" || update.Message.Text != "Нет" && update.Message.Text != "Да" {
				msg := tgbotapi.NewMessage(userID, "Готовы ли вы поделиться результатами своего исследования?")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Да"),
						tgbotapi.NewKeyboardButton("Нет"),
					),
				)
				bot.Send(msg)
				continue
			}
			err = sql.UpdateCustomerByUserId(ctx, models.Customer{
				Id:      userID,
				Results: update.Message.Text,
			})
			if err != nil {
				log.Fatal(err)
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
				continue
			}
			session.CurrentScene = SceneAskDescription
			msg := tgbotapi.NewMessage(userID, "Комментарий(дополнительно):")
			bot.Send(msg)
		case SceneAskUniversity:
			if update.Message.Text != "Экономика" && update.Message.Text != "Психология" && update.Message.Text != "Маркетинг" && update.Message.Text != "Юриспруденция" && update.Message.Text != "Другое" && update.Message.Text != "Неважно" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректный факультет")
				if session.User.IsCustomer == 1 {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Экономика"),
							tgbotapi.NewKeyboardButton("Психология"),
							tgbotapi.NewKeyboardButton("Маркетинг"),
						),
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Юриспруденция"),
							tgbotapi.NewKeyboardButton("Другое"),
							tgbotapi.NewKeyboardButton("Неважно"),
						),
					)
				} else {
					if session.User.IsCustomer == 1 {
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
					}
				}
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
					Id:         userID,
					University: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.University = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:         userID,
					University: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				msg := tgbotapi.NewMessage(userID, "\nВы работаете?")
				msg.ParseMode = "markdown"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Да"),
						tgbotapi.NewKeyboardButton("Нет"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskIsJob
				continue
			}
			s := ""
			if session.User.IsCustomer == 1 {
				sa, err := sql.GetCustomerByUserId(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			} else {
				sa, err := sql.GetRespondentById(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			}
			msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady

		case SceneAskIsJob:
			if update.Message.Text != "Да" && update.Message.Text != "Нет" {
				msg := tgbotapi.NewMessage(userID, "Вы работаете?")
				msg.ParseMode = "markdown"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Да"),
						tgbotapi.NewKeyboardButton("Нет"),
					),
				)
				bot.Send(msg)
				continue
			}
			if update.Message.Text == "Да" {
				msg := tgbotapi.NewMessage(userID, "В какой сфере вы работаете?")
				msg.ParseMode = "markdown"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Бизнес"),
						tgbotapi.NewKeyboardButton("Медицина"),
						tgbotapi.NewKeyboardButton("IT"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Строительство"),
						tgbotapi.NewKeyboardButton("Другое"),
						tgbotapi.NewKeyboardButton("Неважно"),
					),
				)
				bot.Send(msg)
				session.CurrentScene = SceneAskJob
			}

		case SceneAskDifferentUniversity:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:         userID,
					University: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.University = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:         userID,
					University: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			s := ""
			if session.User.IsCustomer == 1 {
				sa, err := sql.GetCustomerByUserId(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			} else {
				sa, err := sql.GetRespondentById(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			}
			msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskJob:
			if update.Message.Text != "Бизнес" && update.Message.Text != "Медицина" && update.Message.Text != "Строительство" && update.Message.Text != "IT" && update.Message.Text != "Другое" {
				msg := tgbotapi.NewMessage(userID, "Выберите корректную сферу")
				if session.User.IsCustomer == 1 {
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Бизнес"),
							tgbotapi.NewKeyboardButton("Медицина"),
							tgbotapi.NewKeyboardButton("IT"),
						),
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Строительство"),
							tgbotapi.NewKeyboardButton("Другое"),
							tgbotapi.NewKeyboardButton("Неважно"),
						),
					)
				} else {
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
				}
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
					Id:  userID,
					Job: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Job = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:  userID,
					Job: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			s := ""
			if session.User.IsCustomer == 1 {
				sa, err := sql.GetCustomerByUserId(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			} else {
				sa, err := sql.GetRespondentById(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			}
			msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskDifferentJob:
			if session.User.IsCustomer == 1 {
				err = sql.UpdateCustomerByUserId(ctx, models.Customer{
					Id:  userID,
					Job: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			} else {
				session.User.Respondent.Job = update.Message.Text
				err = sql.UpdateRespondentByUserId(ctx, models.Respondent{
					Id:  userID,
					Job: update.Message.Text,
				})
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
			}
			s := ""
			if session.User.IsCustomer == 1 {
				sa, err := sql.GetCustomerByUserId(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			} else {
				sa, err := sql.GetRespondentById(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			}
			msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskDescription:
			err = sql.UpdateCustomerByUserId(ctx, models.Customer{
				Id:   userID,
				Desc: update.Message.Text,
			})
			if err != nil {
				log.Fatal(err)
				bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
				continue
			}
			s := ""
			if session.User.IsCustomer == 1 {
				sa, err := sql.GetCustomerByUserId(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			} else {
				sa, err := sql.GetRespondentById(ctx, userID)
				if err != nil {
					log.Fatal(err)
					bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
					continue
				}
				fmt.Println(sa)
				s = sa.ToString()
			}
			msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
			msg.ParseMode = "markdown"
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Да"),
					tgbotapi.NewKeyboardButton("Редактировать"),
				),
			)
			bot.Send(msg)
			session.CurrentScene = SceneAskIsReady
		case SceneAskIsReady:
			if update.Message.Text != "Да" && update.Message.Text != "Редактировать" {
				s := ""
				if session.User.IsCustomer == 0 {
					s = session.User.Customer.ToString()
				} else {
					s = session.User.Respondent.ToString()
				}
				msg := tgbotapi.NewMessage(userID, s+"\nВсе верно?")
				msg.ParseMode = "markdown"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Да"),
						tgbotapi.NewKeyboardButton("Редактировать"),
					),
				)
				bot.Send(msg)
				continue
			}
			if session.User.IsCustomer == 1 {
				if update.Message.Text == "Да" {
					session.User.Customer.Ready = true
					err := sql.SetReadyCustomer(ctx, userID)
					if err != nil {
						log.Fatal(err)
						bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
						continue
					}
					msg := tgbotapi.NewMessage(userID, "Ваше объявление выставлено на доску исследователя.\nВы можете редактировать свою заявку и просматривать откликнувшихся в личном кабинете и меню")
					bot.Send(msg)
					session.CurrentScene = SceneMenu
					continue
				}
			} else {
				if update.Message.Text == "Да" {
					session.User.Respondent.Ready = true
					err := sql.SetReadyRespondent(ctx, userID)
					if err != nil {
						log.Fatal(err)
						bot.Send(tgbotapi.NewMessage(userID, "Ошибка при обновлении данных, попробуйте еще раз"))
						continue
					}
					msg := tgbotapi.NewMessage(userID, "Отлично, теперь вы можете откликаться на заявки на доске исследователя.")
					bot.Send(msg)
					continue
					session.CurrentScene = SceneMenu
				}
			}
			if update.Message.Text == "Редактировать" {
				session.CurrentScene = SceneAskName
				msg := tgbotapi.NewMessage(userID, "Введите имя")
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
