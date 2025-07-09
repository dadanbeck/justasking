package services

import (
	"testing"
)

func TestCalculateNPS(t *testing.T) {
	tests := []struct {
		name        string
		nps         NPS
		expected    int
		expectError bool
	}{
		{
			name: "Basic Case",
			nps: NPS{
				TotalSurvey: 10,
				Promoters:   6,
				Passives:    2,
				Detractors:  2,
			},
			expected:    40,
			expectError: false,
		},
		{
			name: "Zero survey",
			nps: NPS{
				TotalSurvey: 0,
				Promoters:   0,
				Passives:    0,
				Detractors:  0,
			},
			expected:    0,
			expectError: false,
		},
		{
			name: "All Detractors",
			nps: NPS{
				TotalSurvey: 5,
				Promoters:   0,
				Passives:    0,
				Detractors:  5,
			},
			expected:    -100,
			expectError: false,
		},
		{
			name: "All Promoters",
			nps: NPS{
				TotalSurvey: 4,
				Promoters:   4,
				Passives:    0,
				Detractors:  0,
			},
			expected:    100,
			expectError: false,
		},
		{
			name: "Invalid Total Survey",
			nps: NPS{
				TotalSurvey: 10,
				Promoters:   6,
				Passives:    3,
				Detractors:  3,
			},
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.nps.CalculateNPS()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but go nil")
				}
			}

			if got != tt.expected {
				if err != nil {
					t.Errorf("Did not expect error, but got %v", err)
				}

				t.Errorf("CalculateNPS() = %d, want %d", got, tt.expected)
			}
		})
	}
}
