package mcts

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Node represents a state in the MCTS tree
type Node struct {
	sequence     []interface{}
	parent       *Node
	children     []*Node
	visits       int
	totalFitness float64
	mu           sync.Mutex
	unusedMoves  []interface{}
}

// Config holds the MCTS configuration parameters
type Config struct {
	ExplorationConstant  float64
	MaxIterations        int
	TargetSeqLength      int // Set to -1 to use IsSequenceTerminated instead
	RandomSeed           int64
	DebugLevel           int
	IsSequenceTerminated func(sequence []interface{}) bool
	SequenceToString     func(sequence []interface{}) string // New field for custom sequence string conversion
}

type NextElementsFunc func(sequence []interface{}) []interface{}
type FitnessFunc func(sequence []interface{}) float64

// isSequenceComplete checks if the sequence should stop growing
func isSequenceComplete(sequence []interface{}, config Config) bool {
	if config.TargetSeqLength != -1 {
		return len(sequence) >= config.TargetSeqLength
	}
	return config.IsSequenceTerminated != nil && config.IsSequenceTerminated(sequence)
}

// Run executes the MCTS algorithm
func Run(
	initialSequence []interface{},
	nextElements NextElementsFunc,
	fitnessFunc FitnessFunc,
	config Config,
) ([]interface{}, error) {
	if config.ExplorationConstant == 0 {
		config.ExplorationConstant = 1.41
	}

	if config.TargetSeqLength == -1 && config.IsSequenceTerminated == nil {
		return nil, fmt.Errorf("when TargetSeqLength is -1, IsSequenceTerminated function must be provided")
	}

	rand.Seed(config.RandomSeed)
	startTime := time.Now()

	root := &Node{
		sequence:    initialSequence,
		unusedMoves: nextElements(initialSequence),
	}

	var bestSequence []interface{}
	bestFitness := math.MaxFloat64

	// Main MCTS loop
	for i := 0; i < config.MaxIterations; i++ {
		// Selection phase
		selected := selection(root, config.ExplorationConstant, config)

		// Expansion phase
		expanded := expansion(selected, nextElements)
		if expanded == nil {
			continue // Skip if expansion wasn't possible
		}

		// Simulation phase
		simulatedSeq := simulation(expanded, nextElements, config)
		fitness := fitnessFunc(simulatedSeq)

		// Backpropagation phase
		backpropagate(expanded, fitness)

		// Update best found solution
		if isSequenceComplete(simulatedSeq, config) && fitness < bestFitness {
			bestFitness = fitness
			bestSequence = make([]interface{}, len(simulatedSeq))
			copy(bestSequence, simulatedSeq)
		}

		// Progress reporting
		if config.DebugLevel > 0 && i%100 == 0 {
			stats := ProgressStats{
				Iterations:   i + 1,
				BestFitness:  bestFitness,
				BestSequence: bestSequence,
				TreeDepth:    getTreeDepth(root),
				TotalNodes:   countNodes(root),
				Time:         time.Since(startTime),
			}
			printProgress(stats, config)
		}
	}

	// If no valid sequence was found, build one
	if bestSequence == nil {
		bestSequence = buildSequence(initialSequence, nextElements, config)
	}

	return bestSequence, nil
}

func selection(node *Node, explorationConstant float64, config Config) *Node {
	for !isSequenceComplete(node.sequence, config) && len(node.children) > 0 {
		node.mu.Lock()
		var selected *Node
		bestUCT := math.MaxFloat64

		for _, child := range node.children {
			child.mu.Lock()
			uct := calculateUCT(child, explorationConstant)
			child.mu.Unlock()

			if uct < bestUCT {
				bestUCT = uct
				selected = child
			}
		}
		node.mu.Unlock()

		if selected == nil {
			break
		}
		node = selected
	}
	return node
}

// calculateUCT remains unchanged
func calculateUCT(node *Node, explorationConstant float64) float64 {
	if node.visits == 0 {
		return -math.MaxFloat64
	}

	exploitation := node.totalFitness / float64(node.visits)
	exploration := explorationConstant * math.Sqrt(math.Log(float64(node.parent.visits))/float64(node.visits))
	return exploitation - exploration
}

// expansion remains unchanged
func expansion(node *Node, nextElements NextElementsFunc) *Node {
	node.mu.Lock()
	defer node.mu.Unlock()

	if len(node.unusedMoves) == 0 {
		node.unusedMoves = nextElements(node.sequence)
	}

	if len(node.unusedMoves) == 0 {
		return nil
	}

	moveIndex := rand.Intn(len(node.unusedMoves))
	move := node.unusedMoves[moveIndex]

	node.unusedMoves[moveIndex] = node.unusedMoves[len(node.unusedMoves)-1]
	node.unusedMoves = node.unusedMoves[:len(node.unusedMoves)-1]

	newSequence := make([]interface{}, len(node.sequence)+1)
	copy(newSequence, node.sequence)
	newSequence[len(node.sequence)] = move

	child := &Node{
		sequence: newSequence,
		parent:   node,
	}

	node.children = append(node.children, child)
	return child
}

func simulation(node *Node, nextElements NextElementsFunc, config Config) []interface{} {
	sequence := make([]interface{}, len(node.sequence))
	copy(sequence, node.sequence)

	for !isSequenceComplete(sequence, config) {
		moves := nextElements(sequence)
		if len(moves) == 0 {
			break
		}
		move := moves[rand.Intn(len(moves))]
		sequence = append(sequence, move)
	}

	return sequence
}

// backpropagate remains unchanged
func backpropagate(node *Node, fitness float64) {
	for node != nil {
		node.mu.Lock()
		node.visits++
		node.totalFitness += fitness
		node.mu.Unlock()
		node = node.parent
	}
}

// buildSequence updated to use the new termination logic
func buildSequence(initial []interface{}, nextElements NextElementsFunc, config Config) []interface{} {
	sequence := make([]interface{}, len(initial))
	copy(sequence, initial)

	for !isSequenceComplete(sequence, config) {
		moves := nextElements(sequence)
		if len(moves) == 0 {
			break
		}
		sequence = append(sequence, moves[0])
	}

	return sequence
}

// Helper functions remain unchanged...
func getTreeDepth(node *Node) int {
	if len(node.children) == 0 {
		return 0
	}
	maxDepth := 0
	for _, child := range node.children {
		depth := getTreeDepth(child)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth + 1
}

func countNodes(node *Node) int {
	count := 1
	for _, child := range node.children {
		count += countNodes(child)
	}
	return count
}

type ProgressStats struct {
	Iterations   int
	BestFitness  float64
	BestSequence []interface{}
	TreeDepth    int
	TotalNodes   int
	Time         time.Duration
}

func printProgress(stats ProgressStats, config Config) {
	fmt.Printf("\n=== Progress Report (Iteration %d) ===\n", stats.Iterations)
	fmt.Printf("Best Fitness: %f\n", stats.BestFitness)
	fmt.Printf("Time Elapsed: %v\n", stats.Time)

	if config.DebugLevel > 1 {
		fmt.Printf("Tree Depth: %d\n", stats.TreeDepth)
		fmt.Printf("Total Nodes: %d\n", stats.TotalNodes)
		if config.SequenceToString != nil {
			fmt.Printf("Best Sequence: %s\n", config.SequenceToString(stats.BestSequence))
		} else {
			fmt.Printf("Best Sequence: %v\n", stats.BestSequence)
		}
	}
}
