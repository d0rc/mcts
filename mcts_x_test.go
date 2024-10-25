package mcts

import (
	"math"
	"testing"
	"time"
)

type TestProblem struct {
	targetSum     int
	allowedDigits []int
	maxLength     int
}

func (p *TestProblem) nextElements(seq []interface{}) []interface{} {
	if len(seq) >= p.maxLength {
		return nil
	}

	elements := make([]interface{}, len(p.allowedDigits))
	for i, digit := range p.allowedDigits {
		elements[i] = digit
	}
	return elements
}

func (p *TestProblem) fitness(seq []interface{}) float64 {
	if len(seq) != p.maxLength {
		return math.MaxFloat64
	}

	sum := 0
	for _, val := range seq {
		sum += val.(int)
	}
	return math.Pow(float64(sum-p.targetSum), 2)
}

func TestMCTSBasicFunctionality(t *testing.T) {
	problem := &TestProblem{
		targetSum:     15,
		allowedDigits: []int{1, 2, 3, 4, 5},
		maxLength:     4,
	}

	config := Config{
		ExplorationConstant: 2.0, // Increased for more exploration
		MaxIterations:       2000,
		TargetSeqLength:     4,
		RandomSeed:          time.Now().UnixNano(),
		DebugLevel:          0,
	}

	bestSeq, err := Run(
		[]interface{}{},
		problem.nextElements,
		problem.fitness,
		config,
	)

	if err != nil {
		t.Fatalf("MCTS failed with error: %v", err)
	}

	sum := 0
	for _, val := range bestSeq {
		sum += val.(int)
	}

	t.Logf("Target sum: %d", problem.targetSum)
	t.Logf("Best sequence found: %v", bestSeq)
	t.Logf("Sequence sum: %d", sum)
	t.Logf("Fitness (squared error): %f", problem.fitness(bestSeq))

	if len(bestSeq) != config.TargetSeqLength {
		t.Errorf("Expected sequence length %d, got %d", config.TargetSeqLength, len(bestSeq))
	}

	// Check if the sequence is valid
	for _, val := range bestSeq {
		num := val.(int)
		found := false
		for _, allowed := range problem.allowedDigits {
			if num == allowed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Sequence contains invalid number: %d", num)
		}
	}

	// Verify fitness is reasonable
	fitness := problem.fitness(bestSeq)
	if fitness > 100 { // Allow some deviation from target
		t.Errorf("Fitness too high: %f", fitness)
	}
}

// MonotonicTestProblem handles both strictly increasing and non-decreasing sequences
type MonotonicTestProblem struct {
	targetSum      int
	allowedDigits  []int
	maxLength      int
	strictlyStrict bool // true for strictly increasing, false for non-decreasing
}

func (p *MonotonicTestProblem) nextElements(seq []interface{}) []interface{} {
	if len(seq) >= p.maxLength {
		return nil
	}

	// For empty sequence, return all possible digits
	if len(seq) == 0 {
		elements := make([]interface{}, len(p.allowedDigits))
		for i, digit := range p.allowedDigits {
			elements[i] = digit
		}
		return elements
	}

	// Get last element in sequence
	lastNum := seq[len(seq)-1].(int)

	// Return numbers based on monotonicity requirement
	var validMoves []interface{}
	for _, digit := range p.allowedDigits {
		if p.strictlyStrict {
			if digit > lastNum {
				validMoves = append(validMoves, digit)
			}
		} else {
			if digit >= lastNum {
				validMoves = append(validMoves, digit)
			}
		}
	}

	return validMoves
}

func (p *MonotonicTestProblem) fitness(seq []interface{}) float64 {
	if len(seq) != p.maxLength {
		return math.MaxFloat64
	}

	// Check monotonicity
	for i := 1; i < len(seq); i++ {
		if p.strictlyStrict {
			if seq[i].(int) <= seq[i-1].(int) {
				return math.MaxFloat64
			}
		} else {
			if seq[i].(int) < seq[i-1].(int) {
				return math.MaxFloat64
			}
		}
	}

	sum := 0
	for _, val := range seq {
		sum += val.(int)
	}
	return math.Pow(float64(sum-p.targetSum), 2)
}

func TestMCTSMonotonicSequence(t *testing.T) {
	tests := []struct {
		name           string
		targetSum      int
		length         int
		maxDiff        float64
		strictlyStrict bool
	}{
		{
			name:           "Strictly Increasing",
			targetSum:      10,
			length:         3,
			maxDiff:        4,
			strictlyStrict: true,
		},
		{
			name:           "Non-decreasing",
			targetSum:      15,
			length:         4,
			maxDiff:        4,
			strictlyStrict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := &MonotonicTestProblem{
				targetSum:      tt.targetSum,
				allowedDigits:  []int{1, 2, 3, 4, 5},
				maxLength:      tt.length,
				strictlyStrict: tt.strictlyStrict,
			}

			config := Config{
				ExplorationConstant: 4.0,
				MaxIterations:       5000,
				TargetSeqLength:     tt.length,
				RandomSeed:          time.Now().UnixNano(),
				DebugLevel:          0,
			}

			bestSeq, _ := Run(
				[]interface{}{},
				problem.nextElements,
				problem.fitness,
				config,
			)

			// Verify sequence properties
			prev := -1
			sum := 0
			for i, v := range bestSeq {
				num := v.(int)
				sum += num

				if i > 0 {
					if tt.strictlyStrict {
						if num <= prev {
							t.Errorf("Sequence not strictly increasing at position %d: %v", i, bestSeq)
						}
					} else {
						if num < prev {
							t.Errorf("Sequence not non-decreasing at position %d: %v", i, bestSeq)
						}
					}
				}
				prev = num
			}

			diff := math.Abs(float64(sum - tt.targetSum))
			t.Logf("Sequence: %v", bestSeq)
			t.Logf("Sum: %d (target: %d, diff: %f)", sum, tt.targetSum, diff)
			t.Logf("Monotonicity type: %s", map[bool]string{true: "Strictly Increasing", false: "Non-decreasing"}[tt.strictlyStrict])

			if diff > tt.maxDiff {
				t.Errorf("Sum too far from target: got %d, want %d (diff: %f)", sum, tt.targetSum, diff)
			}

			// Additional validation for sequence length
			if len(bestSeq) != tt.length {
				t.Errorf("Wrong sequence length: got %d, want %d", len(bestSeq), tt.length)
			}

			// Validate all numbers are from allowed digits
			for _, v := range bestSeq {
				num := v.(int)
				valid := false
				for _, allowed := range problem.allowedDigits {
					if num == allowed {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("Invalid number in sequence: %d", num)
				}
			}
		})
	}
}

// Additional comprehensive test for edge cases
func TestMCTSMonotonicEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		targetSum      int
		length         int
		allowedDigits  []int
		strictlyStrict bool
		shouldSucceed  bool
	}{
		{
			name:           "Minimal Strictly Increasing",
			targetSum:      3,
			length:         2,
			allowedDigits:  []int{1, 2},
			strictlyStrict: true,
			shouldSucceed:  true,
		},
		{
			name:           "Non-decreasing with Repeats",
			targetSum:      8,
			length:         3,
			allowedDigits:  []int{2, 3, 4},
			strictlyStrict: false,
			shouldSucceed:  true,
		},
		{
			name:           "Impossible Strictly Increasing",
			targetSum:      15,
			length:         4,
			allowedDigits:  []int{1, 2}, // Not enough different numbers
			strictlyStrict: true,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := &MonotonicTestProblem{
				targetSum:      tt.targetSum,
				allowedDigits:  tt.allowedDigits,
				maxLength:      tt.length,
				strictlyStrict: tt.strictlyStrict,
			}

			config := Config{
				ExplorationConstant: 2.0,
				MaxIterations:       1000,
				TargetSeqLength:     tt.length,
				RandomSeed:          time.Now().UnixNano(),
				DebugLevel:          0,
			}

			bestSeq, err := Run(
				[]interface{}{},
				problem.nextElements,
				problem.fitness,
				config,
			)

			// Log results
			t.Logf("Test case: %s", tt.name)
			t.Logf("Sequence found: %v", bestSeq)

			if len(bestSeq) > 0 {
				sum := sequenceSum(bestSeq)
				t.Logf("Sum: %d (target: %d)", sum, tt.targetSum)
			}

			// For impossible cases, we expect either no solution or an invalid one
			if !tt.shouldSucceed {
				if err == nil && problem.fitness(bestSeq) < math.MaxFloat64 {
					t.Errorf("Expected impossible case to fail, but got valid sequence: %v", bestSeq)
				}
				return
			}

			// For possible cases, validate the solution
			if len(bestSeq) != tt.length {
				t.Errorf("Wrong sequence length: got %d, want %d", len(bestSeq), tt.length)
			}

			validateMonotonicSequence(t, bestSeq, tt.strictlyStrict)
		})
	}
}

func validateMonotonicSequence(t *testing.T, seq []interface{}, strictlyStrict bool) {
	if len(seq) < 2 {
		return
	}

	prev := seq[0].(int)
	for i := 1; i < len(seq); i++ {
		curr := seq[i].(int)
		if strictlyStrict {
			if curr <= prev {
				t.Errorf("Sequence not strictly increasing at position %d: %v", i, seq)
			}
		} else {
			if curr < prev {
				t.Errorf("Sequence not non-decreasing at position %d: %v", i, seq)
			}
		}
		prev = curr
	}
}

// Helper function to calculate sequence sum
func sequenceSum(seq []interface{}) int {
	sum := 0
	for _, v := range seq {
		sum += v.(int)
	}
	return sum
}
