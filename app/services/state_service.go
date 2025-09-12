package services

import (
	"encoding/json"
	"gorm.io/gorm"
	"idoctor-bot/app/models"
)

type StateService struct {
	db *gorm.DB
}

func NewStateService(db *gorm.DB) *StateService {
	return &StateService{db: db}
}

// SetState устанавливает состояние пользователя
func (s *StateService) SetState(telegramID int64, state models.UserStateType, data interface{}) error {
	var dataJSON string
	if data != nil {
		if jsonData, err := json.Marshal(data); err == nil {
			dataJSON = string(jsonData)
		}
	}

	return s.db.Where("telegram_id = ?", telegramID).Assign(models.UserState{
		TelegramID: telegramID,
		State:      state,
		Data:       dataJSON,
	}).FirstOrCreate(&models.UserState{}).Error
}

// GetState получает состояние пользователя
func (s *StateService) GetState(telegramID int64) (*models.UserState, error) {
	var state models.UserState
	err := s.db.Where("telegram_id = ?", telegramID).First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Создаем состояние по умолчанию
			state = models.UserState{
				TelegramID: telegramID,
				State:      models.StateIdle,
			}
			return &state, nil
		}
		return nil, err
	}
	return &state, nil
}

// ClearState сбрасывает состояние пользователя в idle
func (s *StateService) ClearState(telegramID int64) error {
	return s.SetState(telegramID, models.StateIdle, nil)
}

// GetStateData получает данные состояния пользователя и десериализует их
func (s *StateService) GetStateData(telegramID int64, data interface{}) error {
	state, err := s.GetState(telegramID)
	if err != nil {
		return err
	}

	if state.Data != "" {
		return json.Unmarshal([]byte(state.Data), data)
	}
	return nil
}
