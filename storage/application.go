package sql

import (
	"context"
	"errors"

	"github.com/slipneff/minor-bot/models"
	"gorm.io/gorm"
)

func (s *Storage) CreateUser(ctx context.Context, user *models.User) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Create(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return err
	}
	return nil
}
func (s *Storage) CreateRespondent(ctx context.Context, app *models.Respondent) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Create(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Storage) CreateCustomer(ctx context.Context, app *models.Customer) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Create(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Storage) GetCustomerByUserId(ctx context.Context, id int64) (*models.Customer, error) {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	var customer models.Customer
	err := tr.Where("user_id = ?", id).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (s *Storage) UpdateCustomerByUserId(ctx context.Context, app models.Customer) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.Customer{}).Where("user_id = ?", app.UserId).Updates(app).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) UpdateRespondentByUserId(ctx context.Context, app models.Respondent) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.Respondent{}).Where("id = ?", app.Id).Updates(app).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetRespondentById(ctx context.Context, id int64) (*models.Respondent, error) {
	var res *models.Respondent
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&res).Where("id = ?", id).Find(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Storage) FindRespondend(ctx context.Context, res models.Respondent) ([]models.Respondent, error) {
	var out []models.Respondent
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&out).Where("age = ? AND gender = ? AND geo = ? AND category = ? and university = ? AND job = ? AND ready = true", res.Age,
		res.Gender, res.Geo, res.Category, res.University, res.Job).Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Storage) GetReadyRespondent(ctx context.Context, id int64) ([]*models.Respondent, error) {
	var respondents []*models.Respondent
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&respondents).Where("ready = ? AND id > ?", true, id).Order("id asc").Limit(2).Find(&respondents).Error
	if err != nil {
		return nil, err
	}
	return respondents, nil
}
func (s *Storage) GetReadyCustomer(ctx context.Context, id int64) ([]*models.Customer, error) {
	var customers []*models.Customer
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&customers).Where("ready = ? AND user_id > ?", true, id).Order("id asc").Limit(2).Find(&customers).Error
	if err != nil {
		return nil, err
	}
	return customers, nil
}
func (s *Storage) MinusOneBalanceUser(ctx context.Context, userId int64) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.User{}).Where("id = ?", userId).UpdateColumn("balance", gorm.Expr("balance - 1")).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) PlusOneBalanceUser(ctx context.Context, userId int64) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.User{}).Where("id = ?", userId).UpdateColumn("balance", gorm.Expr("balance + 1")).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) CreateInterview(ctx context.Context, interview *models.Interview) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Create(interview).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) FindInterviewByCustomerId(ctx context.Context, id int64) ([]models.Interview, error) {
	var interview []models.Interview
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Where("customer_id = ? AND approved_cust = true AND approved_resp = true", id).Find(&interview).Error
	if err != nil {
		return nil, err
	}
	return interview, nil
}
func (s *Storage) GetLastInterviewByCustomer(ctx context.Context, id int64) (*models.Interview, error) {
	var interview *models.Interview
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Where("customer_id = ?", id).Last(&interview).Error
	if err != nil {
		return nil, err
	}
	return interview, nil
}

func (s *Storage) ApproveInterviewByCustomer(ctx context.Context, interview *models.Interview) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(interview).Update("approved_cust", "true").Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) ApproveInterviewByRespondent(ctx context.Context, interview *models.Interview) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(interview).Update("approved_resp", "true").Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) FindInterviewByRespondentId(ctx context.Context, id int64) (*models.Interview, error) {
	var interview models.Interview
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Where("respondent_id = ? AND active = true", id).First(&interview).Error
	if err != nil {
		return nil, err
	}
	return &interview, nil
}

func (s *Storage) DeleteInterviewByRespondentID(ctx context.Context, id int64) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.Interview{}).Delete("respondent_id = ?", id).Error
	if err != nil {
		return err
	}
	return nil
}
func (s *Storage) GetBalanceUser(ctx context.Context, id int64) (int, error) {
	var user models.User
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&user).Where("id = ?", id).Select("balance").Find(&user).Error
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

func (s *Storage) ResetAll(ctx context.Context, id int64) error {
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.User{}).Delete("id = ?", id).Error
	if err != nil {
		return err
	}
	err = tr.Model(&models.Customer{}).Delete("user_id =?", id).Error
	if err != nil {
		return err
	}
	err = tr.Model(&models.Respondent{}).Delete("id =?", id).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetRespondentsByCustomerId(ctx context.Context, id int64) ([]int64, error) {
	var respondents []int64
	tr := s.getter.DefaultTrOrDB(ctx, s.db).WithContext(ctx)
	err := tr.Model(&models.Interview{}).Where("customer_id = ? && approved_resp = true", id).Find(&respondents).Error
	if err != nil {
		return nil, err
	}
	return respondents, nil
}
