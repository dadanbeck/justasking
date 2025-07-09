package services

import "time"

// Holds the answer in every question.
type Answer struct {
	QuestionID string
	Value      any
}

// Contains the collection of answers in every survery.
type Response struct {
	ID        string
	SurveyID  string
	Answers   []Answer
	CreatedAt time.Time
}

// Handles every response for every survey.
//
// This also handles storing the responses to data store for persistence.
type SurveyResponseService interface{}

type surveyResponseServiceImpl struct{}

// Instantiate the `SurveyResponseService`.
func NewSurveyResponseService() SurveyResponseService {
	return &surveyResponseServiceImpl{}
}
