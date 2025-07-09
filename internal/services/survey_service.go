package services

import (
	"errors"
)

// Handles the survey CRUD.
type SurveyService interface {
	GetSurvey(surveyID string) (*Survey, error)
}

type surveyServiceImpl struct{}

// Instantiate the SurveyService.
func NewSurveyService() SurveyService {
	return &surveyServiceImpl{}
}

func (s *surveyServiceImpl) GetSurvey(surveyID string) (*Survey, error) {
	return nil, errors.New("not yet implemented")
}

func (s *surveyServiceImpl) CreateSurvey() (*Survey, error) {
	return nil, errors.New("not yet implemeted")
}
