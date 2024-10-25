package mcts

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"
)

const (
	numAttempts = 1000
	successRate = 0.95 // 95% success rate required
)

type TestResult struct {
	sequence []interface{}
	sum      int
	fitness  float64
	valid    bool
}

func runParallelAttempts(t *testing.T, problem interface{}, config Config) []TestResult {
	results := make([]TestResult, numAttempts)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Create worker pool
	numWorkers := 8
	attemptChan := make(chan int, numAttempts)

	// Feed attempts into channel
	for i := 0; i < numAttempts; i++ {
		attemptChan <- i
	}
	close(attemptChan)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for attemptIdx := range attemptChan {
				// Create new config with unique seed for this attempt
				attemptConfig := config
				attemptConfig.RandomSeed = time.Now().UnixNano() + int64(attemptIdx)
				attemptConfig.DebugLevel = 0 // Disable debug output for parallel runs

				var nextElems NextElementsFunc
				var fitnessFunc FitnessFunc

				switch p := problem.(type) {
				case *TestProblem:
					nextElems = p.nextElements
					fitnessFunc = p.fitness
				case *MonotonicTestProblem:
					nextElems = p.nextElements
					fitnessFunc = p.fitness
				}

				sequence, _ := Run(
					[]interface{}{},
					nextElems,
					fitnessFunc,
					attemptConfig,
				)

				result := TestResult{
					sequence: sequence,
					sum:      sequenceSum(sequence),
					fitness:  fitnessFunc(sequence),
				}
				result.valid = result.fitness < math.MaxFloat64

				mu.Lock()
				results[attemptIdx] = result
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

func analyzeResults(t *testing.T, results []TestResult, expectedProps TestProperties) {
	var successCount, totalSum int
	var minFitness, maxFitness, avgFitness float64
	minFitness = math.MaxFloat64
	successSequences := make([][]interface{}, 0)

	for _, result := range results {
		if result.valid {
			successCount++
			totalSum += result.sum
			successSequences = append(successSequences, result.sequence)

			if result.fitness < minFitness {
				minFitness = result.fitness
			}
			if result.fitness > maxFitness {
				maxFitness = result.fitness
			}
			avgFitness += result.fitness
		}
	}

	if successCount > 0 {
		avgFitness /= float64(successCount)
	}

	successRate := float64(successCount) / float64(numAttempts)

	// Log detailed statistics
	t.Logf("\nTest Statistics (over %d attempts):", numAttempts)
	t.Logf("Success rate: %.2f%% (%d/%d successful attempts)", successRate*100, successCount, numAttempts)
	t.Logf("Fitness statistics:")
	t.Logf("  Min: %.2f", minFitness)
	t.Logf("  Max: %.2f", maxFitness)
	t.Logf("  Avg: %.2f", avgFitness)
	t.Logf("Best sequence found: %v", getBestSequence(successSequences, expectedProps.targetSum))

	// Fail test if success rate is too low
	minRequiredSuccesses := int(float64(numAttempts) * successRate)
	if successCount < minRequiredSuccesses {
		t.Errorf("Success rate too low: got %.2f%%, want at least %.2f%%",
			float64(successCount)/float64(numAttempts)*100,
			successRate*100)
	}
}

func getBestSequence(sequences [][]interface{}, targetSum int) []interface{} {
	if len(sequences) == 0 {
		return nil
	}

	var bestSeq []interface{}
	bestDiff := math.MaxFloat64

	for _, seq := range sequences {
		sum := sequenceSum(seq)
		diff := math.Abs(float64(sum - targetSum))
		if diff < bestDiff {
			bestDiff = diff
			bestSeq = seq
		}
	}

	return bestSeq
}

type TestProperties struct {
	name           string
	targetSum      int
	length         int
	allowedDigits  []int
	maxDiff        float64
	strictlyStrict bool
	shouldSucceed  bool
}

func TestMCTSRobust(t *testing.T) {
	tests := []TestProperties{
		{
			name:          "Basic Sum Problem",
			targetSum:     15,
			length:        4,
			allowedDigits: []int{1, 2, 3, 4, 5},
			maxDiff:       4,
			shouldSucceed: true,
		},
		{
			name:           "Strictly Increasing",
			targetSum:      10,
			length:         3,
			allowedDigits:  []int{1, 2, 3, 4, 5},
			maxDiff:        4,
			strictlyStrict: true,
			shouldSucceed:  true,
		},
		{
			name:           "Non-decreasing",
			targetSum:      15,
			length:         4,
			allowedDigits:  []int{1, 2, 3, 4, 5},
			maxDiff:        4,
			strictlyStrict: false,
			shouldSucceed:  true,
		},
		{
			name:           "Minimal Strictly Increasing",
			targetSum:      3,
			length:         2,
			allowedDigits:  []int{1, 2},
			maxDiff:        1,
			strictlyStrict: true,
			shouldSucceed:  true,
		},
		{
			name:           "Impossible Strictly Increasing",
			targetSum:      15,
			length:         4,
			allowedDigits:  []int{1, 2},
			maxDiff:        4,
			strictlyStrict: true,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var problem interface{}
			if tt.strictlyStrict {
				problem = &MonotonicTestProblem{
					targetSum:      tt.targetSum,
					allowedDigits:  tt.allowedDigits,
					maxLength:      tt.length,
					strictlyStrict: tt.strictlyStrict,
				}
			} else {
				problem = &TestProblem{
					targetSum:     tt.targetSum,
					allowedDigits: tt.allowedDigits,
					maxLength:     tt.length,
				}
			}

			config := Config{
				ExplorationConstant: 2.0,
				MaxIterations:       2000,
				TargetSeqLength:     tt.length,
				RandomSeed:          time.Now().UnixNano(),
				DebugLevel:          0,
			}

			fmt.Printf("\nRunning test case: %s\n", tt.name)
			results := runParallelAttempts(t, problem, config)
			analyzeResults(t, results, tt)
		})
	}
}
