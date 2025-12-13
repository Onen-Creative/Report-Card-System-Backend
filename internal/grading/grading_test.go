package grading

import (
	"testing"
)

func TestPrimaryGrader(t *testing.T) {
	grader := &PrimaryGrader{}

	tests := []struct {
		name     string
		ca       float64
		exam     float64
		caMax    float64
		examMax  float64
		expected string
	}{
		{"Perfect Score", 40, 60, 40, 60, "A"},
		{"Grade A Lower Bound", 32, 48, 40, 60, "A"},
		{"Grade B", 26, 39, 40, 60, "B"},
		{"Grade C", 20, 30, 40, 60, "C"},
		{"Grade D", 14, 21, 40, 60, "D"},
		{"Grade E", 10, 15, 40, 60, "E"},
		{"Zero Score", 0, 0, 40, 60, "E"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grader.ComputeGrade(tt.ca, tt.exam, tt.caMax, tt.examMax)
			if result.FinalGrade != tt.expected {
				t.Errorf("Expected grade %s, got %s. Reason: %s", tt.expected, result.FinalGrade, result.ComputationReason)
			}
		})
	}
}

func TestNCDCGrader(t *testing.T) {
	grader := &NCDCGrader{}

	tests := []struct {
		name     string
		sb       float64
		ext      float64
		sbMax    float64
		extMax   float64
		expected string
	}{
		{"Perfect Score", 100, 100, 100, 100, "A"},
		{"Grade A", 80, 80, 100, 100, "A"},
		{"Grade B", 60, 70, 100, 100, "B"},
		{"Grade C", 50, 50, 100, 100, "C"},
		{"Grade D", 40, 35, 100, 100, "D"},
		{"Grade E", 20, 20, 100, 100, "E"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grader.ComputeGrade(tt.sb, tt.ext, tt.sbMax, tt.extMax)
			if result.FinalGrade != tt.expected {
				t.Errorf("Expected grade %s, got %s. Reason: %s", tt.expected, result.FinalGrade, result.ComputationReason)
			}
		})
	}
}

func TestUACEGrader_MapMarkToCode(t *testing.T) {
	grader := &UACEGrader{}

	tests := []struct {
		marks    float64
		expected int
	}{
		{100, 1},
		{75, 1},
		{74, 2},
		{70, 2},
		{69, 3},
		{65, 3},
		{64, 4},
		{60, 4},
		{59, 5},
		{55, 5},
		{54, 6},
		{50, 6},
		{49, 7},
		{45, 7},
		{44, 8},
		{40, 8},
		{39, 9},
		{0, 9},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			code := grader.MapMarkToCode(tt.marks)
			if code != tt.expected {
				t.Errorf("Mark %.0f: expected code %d, got %d", tt.marks, tt.expected, code)
			}
		})
	}
}

func TestUACEGrader_2Papers(t *testing.T) {
	grader := &UACEGrader{}

	tests := []struct {
		name     string
		papers   []float64
		expected string
	}{
		{"Both Distinction", []float64{80, 85}, "A"},
		{"Grade B", []float64{70, 65}, "B"},
		{"Grade C", []float64{60, 55}, "C"},
		{"Grade D", []float64{55, 50}, "D"},
		{"Grade E", []float64{50, 45}, "E"},
		{"Grade O", []float64{40, 35}, "O"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grader.ComputeGradeFromPapers(tt.papers)
			if result.FinalGrade != tt.expected {
				t.Errorf("Expected grade %s, got %s. Reason: %s", tt.expected, result.FinalGrade, result.ComputationReason)
			}
		})
	}
}

func TestUACEGrader_3Papers(t *testing.T) {
	grader := &UACEGrader{}

	tests := []struct {
		name     string
		papers   []float64
		expected string
	}{
		{"All Distinction", []float64{80, 85, 90}, "A"},
		{"Grade B", []float64{70, 65, 60}, "B"},
		{"Grade C", []float64{60, 55, 50}, "C"},
		{"Grade D", []float64{55, 50, 45}, "D"},
		{"Grade E Normal", []float64{50, 45, 40}, "E"},
		{"Science Exception - E not O", []float64{35, 30, 50}, "E"}, // codes (9,9,7)
		{"Grade O", []float64{35, 30, 25}, "O"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grader.ComputeGradeFromPapers(tt.papers)
			if result.FinalGrade != tt.expected {
				t.Errorf("Expected grade %s, got %s. Reason: %s", tt.expected, result.FinalGrade, result.ComputationReason)
			}
		})
	}
}

func TestUACEGrader_4Papers(t *testing.T) {
	grader := &UACEGrader{}

	tests := []struct {
		name     string
		papers   []float64
		expected string
	}{
		{"All Distinction", []float64{80, 85, 90, 95}, "A"},
		{"Grade B", []float64{70, 65, 60, 55}, "B"},
		{"Grade C", []float64{60, 55, 50, 45}, "C"},
		{"Grade D", []float64{55, 50, 45, 40}, "D"},
		{"Grade E", []float64{50, 45, 40, 35}, "E"},
		{"Grade O", []float64{40, 35, 30, 25}, "O"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grader.ComputeGradeFromPapers(tt.papers)
			if result.FinalGrade != tt.expected {
				t.Errorf("Expected grade %s, got %s. Reason: %s", tt.expected, result.FinalGrade, result.ComputationReason)
			}
		})
	}
}

func TestUACEGrader_EdgeCases(t *testing.T) {
	grader := &UACEGrader{}

	t.Run("Invalid paper count", func(t *testing.T) {
		result := grader.ComputeGradeFromPapers([]float64{80})
		if result.FinalGrade != "F" {
			t.Errorf("Expected F for invalid paper count, got %s", result.FinalGrade)
		}
	})

	t.Run("Boundary sum of 6", func(t *testing.T) {
		result := grader.ComputeGradeFromPapers([]float64{80, 70}) // codes 1,2 sum=3
		if result.FinalGrade != "A" {
			t.Errorf("Expected A, got %s", result.FinalGrade)
		}
	})

	t.Run("Boundary sum of 18", func(t *testing.T) {
		result := grader.ComputeGradeFromPapers([]float64{40, 40}) // codes 8,8 sum=16
		if result.FinalGrade != "E" {
			t.Errorf("Expected E, got %s", result.FinalGrade)
		}
	})
}
