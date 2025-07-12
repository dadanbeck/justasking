package models

type Survey struct {
	ID      string `db:"id" json:"id"`
	Title   string `db:"title" json:"title"`
	StartID string `db:"start_id" json:"start_id"`
}

type Question struct {
	ID           string            `db:"id" json:"id"`
	SurveyID     string            `db:"survey_id" json:"survey_id"`
	Text         string            `db:"text" json:"text"`
	Type         string            `db:"question_type" json:"type"`        // should be a custom type
	Next         string            `db:"next" json:"next"`                 // this is the next question id
	Conditionals []ConditionalNext `db:"conditionals" json:"conditionals"` // this is a jsonb type in database to hold numerous question determined by prev question
}

type ConditionalNext struct {
	Expression string
	NextID     string
}
