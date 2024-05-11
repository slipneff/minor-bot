package main

import (
	"context"
	"fmt"
	"log"
	"time"
	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"github.com/slipneff/minor-bot/config"
	"github.com/slipneff/minor-bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	sql "github.com/slipneff/minor-bot/storage"
)

func main() {
	ctx := context.Background()
	config := config.MustLoadConfig("config.yaml")
	sql := sql.New(sql.MustNewSQLite(config), trmgorm.DefaultCtxGetter)
	bot, err := tgbotapi.NewBotAPI("6547512514:AAHJpIeMLJZMHAnC608UKe16HQlLDjSqJDY")
	if err != nil {
		log.Panic(err)
	}
	customers, err := sql.GetReadyCustomers(ctx)
	if err != nil {
		log.Println(err)
	} else {
		for _, customer := range customers {

			newSimilarUsers := make(chan models.Respondent)
			go func() {
				for {
					users, err := sql.FindRespondend(ctx, models.Respondent{
						Age:        customer.Age,
						Gender:     customer.Gender,
						Geo:        customer.Geo,
						Category:   customer.Category,
						University: customer.University,
						Job:        customer.Job,
						Ready:      true,
					})
					if err != nil {
						log.Println("Error finding new similar users:", err)
						continue
					}

					for _, user := range users {
						newSimilarUsers <- user
					}

					// Пауза перед следующей проверкой
					time.Sleep(1 * time.Second) // Например, проверка каждый час
				}
			}()

			for newUser := range newSimilarUsers {
				msg := tgbotapi.NewMessage(customer.UserId, fmt.Sprintf("Нашелся клиент, готовый к интервью! Вот его анкета:%s\n", newUser.ToString()))
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Согласен"),
						tgbotapi.NewKeyboardButton("Отказать"),
					),
				)
				bot.Send(msg)
				err := sql.CreateInterview(ctx, &models.Interview{
					CustomerId:    customer.UserId,
					ApplicationId: customer.Id,
					RespondentId:  newUser.Id,
				})
				if err != nil {
					log.Println("Error creating interview:", err)
					continue
				}

				msg = tgbotapi.NewMessage(newUser.Id, "Прислали вашу анкету интервьюеру! Ожидайте ответа.")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Согласен"),
						tgbotapi.NewKeyboardButton("Отказать"),
					),
				)
				bot.Send(msg)
				// session := sessions[customer.UserId]
				// session.CurrentScene = SceneApproveInterviewCustomer
				// sessions[customer.UserId] = session
				// session = sessions[newUser.Id]
				// session.CurrentScene = SceneApproveInterviewRespondent
				// sessions[newUser.Id] = session
			}
		}
	}
}
