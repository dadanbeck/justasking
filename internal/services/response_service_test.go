package services

import (
	"strings"
	"testing"
)

func TestAnswerQuestion_WithValidConditionalMatch(t *testing.T) {
	svc := NewSurveyService()
	responseservice := NewSurveyResponseService(svc)

	question := Question{
		ID:   "q1",
		Text: "Are you over 18?",
		Type: MultipleChoice,
		Options: []string{
			"yes",
			"no",
		},
		Conditionals: []ConditionalNext{
			{
				Expression: `q1 == "yes"`,
				NextID:     "q2",
			},
			{
				Expression: `q1 == "no"`,
				NextID:     "q3",
			},
		},
	}

	nextQuestion := Question{
		ID:   "q2",
		Text: "What is your occupation?",
		Type: Text,
	}

	alternativeNextQuestion := Question{
		ID:   "q3",
		Text: "Are you studying?",
		Options: []string{
			"yes",
			"no",
		},
	}

	survey := Survey{
		ID:      "s1",
		Title:   "Age Check",
		StartID: "q1",
		Questions: map[string]Question{
			"q1": question,
			"q2": nextQuestion,
			"q3": alternativeNextQuestion,
		},
	}

	session := &SurveySession{
		ID:        "sess1",
		SurveyID:  "s1",
		Answers:   make(map[string]any),
		CurrentID: "q1",
	}

	q, err := responseservice.AnswerQuestion(session, "q1", "yes", survey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil || q.ID != "q2" {
		t.Errorf("expected next question to be q2, got %v", q)
	}
	if session.CurrentID != "q2" {
		t.Errorf("expected session.CurrentID to be q2, got %s", session.CurrentID)
	}
	if !session.Completed {
		t.Errorf("expected session to be marked completed")
	}
}

func TestAnswerQuestion_AlternativeMatch(t *testing.T) {
	svc := NewSurveyService()
	responseservice := NewSurveyResponseService(svc)

	question := Question{
		ID:   "q1",
		Text: "Are you over 18?",
		Type: MultipleChoice,
		Options: []string{
			"yes",
			"no",
		},
		Conditionals: []ConditionalNext{
			{
				Expression: `q1 == "yes"`,
				NextID:     "q2",
			},
			{
				Expression: `q1 == "no"`,
				NextID:     "q3",
			},
		},
	}

	nextQuestion := Question{
		ID:   "q2",
		Text: "What is your occupation?",
		Type: Text,
	}

	alternativeNextQuestion := Question{
		ID:   "q3",
		Text: "Are you studying?",
		Options: []string{
			"yes",
			"no",
		},
	}

	survey := Survey{
		ID:      "s1",
		Title:   "Age Check",
		StartID: "q1",
		Questions: map[string]Question{
			"q1": question,
			"q2": nextQuestion,
			"q3": alternativeNextQuestion,
		},
	}

	session := &SurveySession{
		ID:        "sess1",
		SurveyID:  "s1",
		Answers:   make(map[string]any),
		CurrentID: "q1",
	}

	q, err := responseservice.AnswerQuestion(session, "q1", "no", survey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil || q.ID != "q3" {
		t.Errorf("expected next question to be q3, got %v", q)
	}
	if session.CurrentID != "q3" {
		t.Errorf("expected session.CurrentID to be q3, got %s", session.CurrentID)
	}
	if !session.Completed {
		t.Errorf("expected session to be marked completed")
	}
}

func TestAnswerQuestion_InvalidQuestion(t *testing.T) {
	svc := NewSurveyService()
	responseservice := NewSurveyResponseService(svc)
	survey := Survey{
		ID:        "s1",
		Questions: map[string]Question{},
	}
	session := &SurveySession{
		ID:        "sess1",
		SurveyID:  "s1",
		Answers:   make(map[string]any),
		CurrentID: "q999",
	}

	q, err := responseservice.AnswerQuestion(session, "q999", "anything", survey)

	if err == nil || !strings.Contains(err.Error(), "invalid question") {
		t.Errorf("expected nil 'invalid question' error, got %v", err)
	}

	if q != nil {
		t.Errorf("expected nil question, got %+v", q)
	}
}

func TestAnswerQuestion_NoConditionalMatch(t *testing.T) {
	svc := NewSurveyService()
	responseservice := NewSurveyResponseService(svc)

	q1 := Question{
		ID:   "q1",
		Text: "Continue?",
		Type: MultipleChoice,
		Conditionals: []ConditionalNext{
			{
				Expression: `q1 == "yes"`,
				NextID:     "q2",
			},
		},
	}

	survey := Survey{
		ID:      "s1",
		StartID: "q1",
		Questions: map[string]Question{
			"q1": q1,
		},
	}

	session := &SurveySession{
		ID:        "sess1",
		SurveyID:  "s1",
		Answers:   make(map[string]any),
		CurrentID: "q1",
	}

	q, err := responseservice.AnswerQuestion(session, "q1", "no", survey)
	if err == nil || !strings.Contains(err.Error(), "next question not found") {
		t.Errorf("expected 'next question not found' error, got %v", err)
	}
	if q != nil {
		t.Errorf("expected nil question, got %+v", q)
	}
	if session.Completed {
		t.Errorf("expected session to not be completed")
	}
}

func TestAnswerQuestion_AlreadyCompleted(t *testing.T) {
	svc := NewSurveyService()
	responseservice := NewSurveyResponseService(svc)

	survey := Survey{
		ID:        "s1",
		Questions: map[string]Question{},
	}
	session := &SurveySession{
		ID:        "sess1",
		SurveyID:  "s1",
		Completed: true,
	}

	q, err := responseservice.AnswerQuestion(session, "q1", "answer", survey)
	if err == nil || !strings.Contains(err.Error(), "survey already completed") {
		t.Errorf("expected 'survey already completed' error, got %v", err)
	}
	if q != nil {
		t.Errorf("expected nil question, got %+v", q)
	}
}

func TestEvaluateExpression_InvalidSyntax(t *testing.T) {
	result, err := evaluateExpression(`input["q1" == true`, map[string]any{"q1": "yes"})
	if err == nil {
		t.Errorf("expected error for invalid expression, got none")
	}
	if result {
		t.Errorf("expected result to be false, got true")
	}
}

func TestEvaluateExpression_NonBooleanResult(t *testing.T) {
	result, err := evaluateExpression(`q1`, map[string]any{"q1": "hello"})
	if err == nil || !strings.Contains(err.Error(), "expression did not return a boolean") {
		t.Errorf("expected 'expression did not return a boolean' error, got %v", err)
	}
	if result {
		t.Errorf("expected result to be false, got true")
	}
}
