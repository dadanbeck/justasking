package services

import (
	"errors"

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

// TODO: do we have to implement separate service for storing survey and their state to
// make the `SurveyService` works for the answering workflow only.

// Handles the survey flow.
type SurveyService interface {
	AnswerQuestion(session *SurveySession, questionID string, answer any, survey Survey) (*Question, error)
	GetNextQuestionWithLogic(question Question, input map[string]any) (string, error)
}

type surveyServiceImpl struct{}

// Instantiate the SurveyService.
func NewSurveyService() SurveyService {
	return &surveyServiceImpl{}
}

func (s *surveyServiceImpl) AnswerQuestion(session *SurveySession, questionID string, answer any, survey Survey) (*Question, error) {
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

func (s *surveyServiceImpl) GetNextQuestionWithLogic(question Question, input map[string]any) (string, error) {
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
