package output

import "testing"

func TestComputeMetrics_RegressionMultiplier(t *testing.T) {
	m := computeMetrics(100, 320)
	// 320/100 = 3.2
	if m.multiplier != 3.2 {
		t.Errorf("Expected multiplier 3.2, got %.2f", m.multiplier)
	}
}

func TestComputeMetrics_RegressionPercentage(t *testing.T) {
	m := computeMetrics(100, 320)
	// (320-100)/100*100 = 220%
	if m.percentage != 220 {
		t.Errorf("Expected percentage 220, got %.2f", m.percentage)
	}
}

func TestComputeMetrics_ImprovementDetected(t *testing.T) {
	m := computeMetrics(200, 140)
	if m.isRegression {
		t.Error("Expected improvement, got regression")
	}
	if m.percentage >= 0 {
		t.Errorf("Expected negative percentage for improvement, got %.2f", m.percentage)
	}
}

func TestComputeMetrics_DeltaCalculated(t *testing.T) {
	m := computeMetrics(100, 320)
	if m.delta != 220 {
		t.Errorf("Expected delta 220, got %d", m.delta)
	}
}

func TestComputeMetrics_ZeroBaselineReturnsEmpty(t *testing.T) {
	m := computeMetrics(0, 320)
	if m.multiplier != 0 {
		t.Error("Expected zero multiplier for zero baseline")
	}
	if m.percentage != 0 {
		t.Error("Expected zero percentage for zero baseline")
	}
}

func TestComputeMetrics_MultiplierRoundedToOneDecimal(t *testing.T) {
	// 319/100 = 3.19 → rounds to 3.2
	m := computeMetrics(100, 319)
	if m.multiplier != 3.2 {
		t.Errorf("Expected 3.2 after rounding, got %.2f", m.multiplier)
	}
}