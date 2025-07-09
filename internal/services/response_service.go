package services

import (
	"errors"
	"time"

	"github.com/expr-lang/expr"
)

// The type of question being asked.
type QuestionType int

const (
	Text QuestionType = iota + 1
	MultipleChoice
)

// The conditional next question after a question answered
//
// Basically it determines what question will be ask based
// on the answer from prev question.
type ConditionalNext struct {
	Expression string
	NextID     string
}

// The Question object.
type Question struct {
	ID           string
	Text         string
	Type         QuestionType // could be an enum
	Options      []string
	Next         map[string]string
	Conditionals []ConditionalNext
}

// The Survery object.
type Survey struct {
	ID        string
	Title     string
	StartID   string
	Questions map[string]Question // keyed by Question.ID
}

// Holds the state of the current state of the respondent on the survey.
//
// Typically what question they are when they paused or leave it.
type SurveySession struct {
	ID        string
	SurveyID  string
	Answers   map[string]any
	CurrentID string
	Completed bool
}

// Holds the answer in every question.
type Answer struct {
	QuestionID string
	Value      any
}

// Contains the collection of answers in every survery.
type SurveyResponse struct {
	ID        string
	SurveyID  string
	Answers   []Answer
	CreatedAt time.Time
}

// Handles every response for every survey.
type SurveyResponseService interface {
	// Save response
	SaveResponse(response SurveyResponse) error
	// Answers the question.
	AnswerQuestion(session *SurveySession, questionID string, answer any, survey Survey) (*Question, error)
	// Determines what is the next question.
	GetNextQuestionWithLogic(question Question, input map[string]any) (string, error)
}

type surveyResponseServiceImpl struct {
	surveyservice SurveyService
}

// Instantiate the `SurveyResponseService`.
func NewSurveyResponseService(surveyservice SurveyService) SurveyResponseService {
	return &surveyResponseServiceImpl{surveyservice: surveyservice}
}

func (s *surveyResponseServiceImpl) SaveResponse(response SurveyResponse) error {
	_, err := s.surveyservice.GetSurvey(response.SurveyID)
	if err != nil {
		return err
	}

	// we answer here using the survey service

	return errors.New("not yet implemented")
}

func (s *surveyResponseServiceImpl) AnswerQuestion(session *SurveySession, questionID string, answer any, survey Survey) (*Question, error) {
	if session.Completed {
		return nil, errors.New("survery already completed")
	}

	session.Answers[questionID] = answer

	current, ok := survey.Questions[questionID]
	if !ok {
		return nil, errors.New("invalid question")
	}

	input := session.Answers

	nextQuestionID, err := s.GetNextQuestionWithLogic(current, input)
	if err != nil {
		return nil, err
	}

	nextQuestion, exists := survey.Questions[nextQuestionID]

	if !exists {
		session.Completed = false
		session.CurrentID = ""
		return nil, errors.New("next question not found")
	}

	session.Completed = true
	session.CurrentID = nextQuestion.ID
	return &nextQuestion, nil
}

func (s *surveyResponseServiceImpl) GetNextQuestionWithLogic(question Question, input map[string]any) (string, error) {
	for _, cond := range question.Conditionals {
		match, err := evaluateExpression(cond.Expression, input)
		if err != nil {
			return "", err
		}

		if match {
			return cond.NextID, nil
		}
	}

	return "", nil
}

func evaluateExpression(expression string, input map[string]any) (bool, error) {
	program, err := expr.Compile(expression, expr.Env(input))
	if err != nil {
		return false, err
	}

	output, err := expr.Run(program, input)
	if err != nil {
		return false, err
	}

	result, ok := output.(bool)

	if !ok {
		return false, errors.New("expression did not return a boolean")
	}

	return result, nil
}

// TODO: do we have to implement separate service for storing survey and their state to
// make the `SurveyResponseService` works for the answering workflow only.
