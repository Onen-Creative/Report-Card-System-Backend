package grading

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

const (
	RuleVersionPrimary   = "PRIMARY_V1"
	RuleVersionNCDC      = "NCDC_V1"
	RuleVersionUACE      = "UACE_V1"
)

// GradeResult holds computed grade information
type GradeResult struct {
	FinalGrade        string
	ComputationReason string
	RuleVersionHash   string
	PaperCodes        map[string]int // For UACE
}

// PrimaryGrader implements P4-P7 grading
type PrimaryGrader struct{}

func (g *PrimaryGrader) ComputeGrade(caMarks, examMarks, caMax, examMax float64) GradeResult {
	caPercent := (caMarks / caMax) * 40
	examPercent := (examMarks / examMax) * 60
	total := caPercent + examPercent

	grade := ""
	switch {
	case total >= 80:
		grade = "A"
	case total >= 65:
		grade = "B"
	case total >= 50:
		grade = "C"
	case total >= 35:
		grade = "D"
	default:
		grade = "E"
	}

	reason := fmt.Sprintf("CA: %.2f/%.0f (40%%) = %.2f, Exam: %.2f/%.0f (60%%) = %.2f, Total: %.2f → Grade %s",
		caMarks, caMax, caPercent, examMarks, examMax, examPercent, total, grade)

	return GradeResult{
		FinalGrade:        grade,
		ComputationReason: reason,
		RuleVersionHash:   hashRuleVersion(RuleVersionPrimary),
	}
}

// NCDCGrader implements Lower Secondary grading
type NCDCGrader struct{}

func (g *NCDCGrader) ComputeGrade(schoolBasedMarks, externalMarks, schoolBasedMax, externalMax float64) GradeResult {
	sbPercent := (schoolBasedMarks / schoolBasedMax) * 20
	extPercent := (externalMarks / externalMax) * 80
	total := sbPercent + extPercent

	grade := ""
	switch {
	case total >= 80:
		grade = "A"
	case total >= 65:
		grade = "B"
	case total >= 50:
		grade = "C"
	case total >= 35:
		grade = "D"
	default:
		grade = "E"
	}

	reason := fmt.Sprintf("School-Based: %.2f/%.0f (20%%) = %.2f, External: %.2f/%.0f (80%%) = %.2f, Total: %.2f → Grade %s",
		schoolBasedMarks, schoolBasedMax, sbPercent, externalMarks, externalMax, extPercent, total, grade)

	return GradeResult{
		FinalGrade:        grade,
		ComputationReason: reason,
		RuleVersionHash:   hashRuleVersion(RuleVersionNCDC),
	}
}

// UACEGrader implements UACE/UNEB grading
type UACEGrader struct{}

// MapMarkToCode converts 0-100 marks to UNEB code 1-9
func (g *UACEGrader) MapMarkToCode(marks float64) int {
	switch {
	case marks >= 75:
		return 1
	case marks >= 70:
		return 2
	case marks >= 65:
		return 3
	case marks >= 60:
		return 4
	case marks >= 55:
		return 5
	case marks >= 50:
		return 6
	case marks >= 45:
		return 7
	case marks >= 40:
		return 8
	default:
		return 9
	}
}

// ComputeGradeFromPapers computes final grade from paper marks
func (g *UACEGrader) ComputeGradeFromPapers(paperMarks []float64) GradeResult {
	numPapers := len(paperMarks)
	if numPapers < 2 || numPapers > 4 {
		return GradeResult{
			FinalGrade:        "F",
			ComputationReason: fmt.Sprintf("Invalid number of papers: %d", numPapers),
			RuleVersionHash:   hashRuleVersion(RuleVersionUACE),
		}
	}

	// Convert marks to codes
	codes := make([]int, numPapers)
	paperCodes := make(map[string]int)
	for i, mark := range paperMarks {
		codes[i] = g.MapMarkToCode(mark)
		paperCodes[fmt.Sprintf("Paper%d", i+1)] = codes[i]
	}

	// Sort codes ascending for processing
	sortedCodes := make([]int, len(codes))
	copy(sortedCodes, codes)
	sort.Ints(sortedCodes)

	var finalGrade string
	var reason string

	switch numPapers {
	case 2:
		finalGrade, reason = g.compute2Papers(sortedCodes)
	case 3:
		finalGrade, reason = g.compute3Papers(sortedCodes)
	case 4:
		finalGrade, reason = g.compute4Papers(sortedCodes)
	}

	return GradeResult{
		FinalGrade:        finalGrade,
		ComputationReason: fmt.Sprintf("Papers: %v → Codes: %v → %s", paperMarks, codes, reason),
		RuleVersionHash:   hashRuleVersion(RuleVersionUACE),
		PaperCodes:        paperCodes,
	}
}

func (g *UACEGrader) compute2Papers(codes []int) (string, string) {
	sum := codes[0] + codes[1]
	
	switch {
	case sum <= 6:
		return "A", fmt.Sprintf("Sum %d ≤ 6", sum)
	case sum <= 10:
		return "B", fmt.Sprintf("Sum %d ≤ 10", sum)
	case sum <= 12:
		return "C", fmt.Sprintf("Sum %d ≤ 12", sum)
	case sum <= 15:
		return "D", fmt.Sprintf("Sum %d ≤ 15", sum)
	case sum <= 18:
		return "E", fmt.Sprintf("Sum %d ≤ 18", sum)
	default:
		return "O", fmt.Sprintf("Sum %d > 18", sum)
	}
}

func (g *UACEGrader) compute3Papers(codes []int) (string, string) {
	// Best 2 papers
	best2Sum := codes[0] + codes[1]
	
	// Science exception: if best 2 sum is (9,9,X) where X≤7, grade is E not O
	if codes[0] == 9 && codes[1] == 9 && codes[2] <= 7 {
		return "E", fmt.Sprintf("Science exception: codes %v, best 2 sum %d but third ≤7", codes, best2Sum)
	}
	
	switch {
	case best2Sum <= 6:
		return "A", fmt.Sprintf("Best 2 sum %d ≤ 6", best2Sum)
	case best2Sum <= 10:
		return "B", fmt.Sprintf("Best 2 sum %d ≤ 10", best2Sum)
	case best2Sum <= 12:
		return "C", fmt.Sprintf("Best 2 sum %d ≤ 12", best2Sum)
	case best2Sum <= 15:
		return "D", fmt.Sprintf("Best 2 sum %d ≤ 15", best2Sum)
	case best2Sum <= 18:
		return "E", fmt.Sprintf("Best 2 sum %d ≤ 18", best2Sum)
	default:
		return "O", fmt.Sprintf("Best 2 sum %d > 18", best2Sum)
	}
}

func (g *UACEGrader) compute4Papers(codes []int) (string, string) {
	// Best 2 papers
	best2Sum := codes[0] + codes[1]
	
	switch {
	case best2Sum <= 6:
		return "A", fmt.Sprintf("Best 2 sum %d ≤ 6", best2Sum)
	case best2Sum <= 10:
		return "B", fmt.Sprintf("Best 2 sum %d ≤ 10", best2Sum)
	case best2Sum <= 12:
		return "C", fmt.Sprintf("Best 2 sum %d ≤ 12", best2Sum)
	case best2Sum <= 15:
		return "D", fmt.Sprintf("Best 2 sum %d ≤ 15", best2Sum)
	case best2Sum <= 18:
		return "E", fmt.Sprintf("Best 2 sum %d ≤ 18", best2Sum)
	default:
		return "O", fmt.Sprintf("Best 2 sum %d > 18", best2Sum)
	}
}

func hashRuleVersion(version string) string {
	hash := sha256.Sum256([]byte(version))
	return fmt.Sprintf("%x", hash[:8])
}

// GetGrader returns appropriate grader for level
func GetGrader(level string) interface{} {
	switch level {
	case "P4", "P5", "P6", "P7":
		return &PrimaryGrader{}
	case "S1", "S2", "S3", "S4":
		return &NCDCGrader{}
	case "S5", "S6":
		return &UACEGrader{}
	default:
		return nil
	}
}
