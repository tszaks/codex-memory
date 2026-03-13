package analysis

type Confidence struct {
	Level   string   `json:"level"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons"`
}

func buildConfidence(hasRisk bool, structuralLinks int, suggestedTests int, blastRadius int) Confidence {
	score := 0
	reasons := make([]string, 0, 4)

	if hasRisk {
		score += 3
		reasons = append(reasons, "Indexed history exists for this file.")
	} else {
		reasons = append(reasons, "This file is new or outside indexed history.")
	}

	if structuralLinks > 0 {
		score += 2
		reasons = append(reasons, "Structural links were found in the repo.")
	}

	if suggestedTests > 0 {
		score += 2
		reasons = append(reasons, "Focused tests were inferred for this file.")
	}

	if blastRadius > 0 {
		score += 1
		reasons = append(reasons, "The tool found likely impact files.")
	}

	level := "low"
	switch {
	case score >= 6:
		level = "high"
	case score >= 3:
		level = "medium"
	}

	return Confidence{
		Level:   level,
		Score:   score,
		Reasons: reasons,
	}
}
