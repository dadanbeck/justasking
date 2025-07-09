package services

import "fmt"

// NOTE: the formula for determining the NPS
// NPS = %Promoters − %Detractors

type NPS struct {
	// The total of surveyed people
	TotalSurvey int
	// Ratings 9 or 10
	Promoters int
	// Ratings 7 or 8
	Passives int
	// 6 or lower
	Detractors int
}

func (n *NPS) CalculateNPS() (int, error) {
	if n.TotalSurvey == 0 {
		return 0, nil
	}

	// what if the total of the promter, passives and detractors are greater than the total survey.
	totalEntities := (n.Promoters + n.Passives + n.Detractors)
	if n.TotalSurvey < totalEntities {
		return 0, fmt.Errorf("cannot compute nps with total survey is less than from the total of entities: %d total < total entities: %d", n.TotalSurvey, totalEntities)
	}

	promoterCalc := (float64(n.Promoters) / float64(n.TotalSurvey)) * 100
	detractorCalc := (float64(n.Detractors) / float64(n.TotalSurvey)) * 100

	return int(promoterCalc - detractorCalc), nil
}
