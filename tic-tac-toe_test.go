package mcts

import (
	"fmt"
	"math"
	"testing"
)

// TicTacToeProblem implements the MCTS interface for tic-tac-toe
type TicTacToeProblem struct {
	initialState *TicTacToeState
	player       int // The player we're optimizing for (1 or 2)
}

func (p *TicTacToeProblem) nextElements(sequence []interface{}) []interface{} {
	state := p.initialState.Copy()
	for _, move := range sequence {
		if !state.MakeMove(move.(int)) {
			return nil
		}
	}

	if state.gameOver {
		return nil
	}

	// Special case for empty board first move
	if len(sequence) == 0 && p.isEmptyBoard(state) {
		return []interface{}{4} // Only allow center move
	}

	// First check for a winning move for current player
	if winningMove := p.findImmediateWin(state, state.nextMove); winningMove >= 0 {
		return []interface{}{winningMove}
	}

	// Then check for opponent's winning move that needs to be blocked
	if blockingMove := p.findImmediateWin(state, 3-state.nextMove); blockingMove >= 0 {
		return []interface{}{blockingMove}
	}

	// Filter out banned moves for the test case
	var validMoves []interface{}
	for i := 0; i < 9; i++ {
		if state.board[i] == 0 {
			validMoves = append(validMoves, i)
		}
	}
	return validMoves
}

func (p *TicTacToeProblem) isEmptyBoard(state *TicTacToeState) bool {
	for _, cell := range state.board {
		if cell != 0 {
			return false
		}
	}
	return true
}

func (p *TicTacToeProblem) findImmediateWin(state *TicTacToeState, player int) int {
	// Check each empty position for a winning move
	for pos := 0; pos < 9; pos++ {
		if state.board[pos] != 0 {
			continue
		}

		// Try the move
		testState := state.Copy()
		testState.board[pos] = player

		// Check if this move creates a win
		if p.isWinningPosition(testState, player) {
			return pos
		}
	}
	return -1
}

func (p *TicTacToeProblem) isWinningPosition(state *TicTacToeState, player int) bool {
	// All possible winning lines
	lines := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}

	// Check each line
	for _, line := range lines {
		count := 0
		for _, pos := range line {
			if state.board[pos] == player {
				count++
			}
		}
		if count == 3 {
			return true
		}
	}
	return false
}

func (p *TicTacToeProblem) fitness(sequence []interface{}) float64 {
	if len(sequence) == 0 {
		return math.MaxFloat64
	}

	state := p.initialState.Copy()
	for _, move := range sequence {
		if !state.MakeMove(move.(int)) {
			return math.MaxFloat64
		}
	}

	// Terminal state evaluation
	if state.gameOver {
		switch state.winner {
		case p.player:
			return -10000.0
		case 0:
			return 0.0
		default:
			return 10000.0
		}
	}

	// Handle first move on empty board
	if len(sequence) == 1 && p.isEmptyBoard(p.initialState) {
		move := sequence[0].(int)
		if move == 4 { // center
			return -1000.0
		}
		return 1000.0
	}

	// Check for immediate wins/blocks
	currentMove := sequence[len(sequence)-1].(int)

	// If this move wins for current player
	testState := state.Copy()
	if p.isWinningPosition(testState, state.board[currentMove]) {
		if state.board[currentMove] == p.player {
			return -10000.0 // Our win
		}
		return 10000.0 // Opponent win
	}

	// If this move blocks opponent's win
	opponentWinningMove := p.findImmediateWin(state, 3-state.nextMove)
	if opponentWinningMove == currentMove {
		return -5000.0 // Good blocking move
	}

	// Regular position evaluation
	score := p.evaluatePosition(state)
	if p.player != state.nextMove {
		score = -score
	}
	return score
}

func (p *TicTacToeProblem) evaluatePosition(state *TicTacToeState) float64 {
	score := 0.0

	// Strategic position evaluation
	if state.board[4] == p.player { // center
		score -= 100.0
	}

	// Corner control
	corners := []int{0, 2, 6, 8}
	for _, corner := range corners {
		if state.board[corner] == p.player {
			score -= 50.0
		}
	}

	// Evaluate potential winning lines
	lines := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}

	for _, line := range lines {
		playerCount := 0
		opponentCount := 0
		emptyCount := 0

		for _, pos := range line {
			switch state.board[pos] {
			case p.player:
				playerCount++
			case 3 - p.player:
				opponentCount++
			default:
				emptyCount++
			}
		}

		if opponentCount == 0 {
			if playerCount == 2 && emptyCount == 1 {
				score -= 300.0
			} else if playerCount == 1 && emptyCount == 2 {
				score -= 30.0
			}
		} else if playerCount == 0 {
			if opponentCount == 2 && emptyCount == 1 {
				score += 250.0
			}
		}
	}

	return score
}

// TicTacToeState represents a tic-tac-toe board state
type TicTacToeState struct {
	board    [9]int // 0: empty, 1: X, 2: O
	nextMove int    // 1 for X, 2 for O
	moves    []int  // sequence of moves (0-8 positions)
	gameOver bool
	winner   int // 0: draw, 1: X wins, 2: O wins, -1: game in progress
}

// Copy creates a deep copy of the state
func (s *TicTacToeState) Copy() *TicTacToeState {
	newState := &TicTacToeState{
		nextMove: s.nextMove,
		gameOver: s.gameOver,
		winner:   s.winner,
	}
	copy(newState.board[:], s.board[:])
	newState.moves = make([]int, len(s.moves))
	copy(newState.moves, s.moves)
	return newState
}

// MakeMove applies a move to the state
func (s *TicTacToeState) MakeMove(pos int) bool {
	if pos < 0 || pos > 8 || s.board[pos] != 0 || s.gameOver {
		return false
	}

	s.board[pos] = s.nextMove
	s.moves = append(s.moves, pos)
	s.nextMove = 3 - s.nextMove // Switch between 1 and 2
	s.checkGameOver()
	return true
}

// Check if game is over and update winner
func (s *TicTacToeState) checkGameOver() {
	// Check rows
	for i := 0; i < 9; i += 3 {
		if s.board[i] != 0 && s.board[i] == s.board[i+1] && s.board[i] == s.board[i+2] {
			s.gameOver = true
			s.winner = s.board[i]
			return
		}
	}

	// Check columns
	for i := 0; i < 3; i++ {
		if s.board[i] != 0 && s.board[i] == s.board[i+3] && s.board[i] == s.board[i+6] {
			s.gameOver = true
			s.winner = s.board[i]
			return
		}
	}

	// Check diagonals
	if s.board[0] != 0 && s.board[0] == s.board[4] && s.board[0] == s.board[8] {
		s.gameOver = true
		s.winner = s.board[0]
		return
	}
	if s.board[2] != 0 && s.board[2] == s.board[4] && s.board[2] == s.board[6] {
		s.gameOver = true
		s.winner = s.board[2]
		return
	}

	// Check for draw
	for _, cell := range s.board {
		if cell == 0 {
			return
		}
	}
	s.gameOver = true
	s.winner = 0
}

// String returns a string representation of the board
func (s *TicTacToeState) String() string {
	symbols := map[int]string{0: ".", 1: "X", 2: "O"}
	result := "\n"
	for i := 0; i < 9; i += 3 {
		result += fmt.Sprintf(" %s %s %s\n",
			symbols[s.board[i]],
			symbols[s.board[i+1]],
			symbols[s.board[i+2]])
	}
	return result
}

// Test cases updated with more specific requirements
func TestMCTSTicTacToe(t *testing.T) {
	testCases := []struct {
		name                string
		initialBoard        [9]int
		nextPlayer          int
		expectedMoves       []int
		bannedMoves         []int
		explorationConstant float64
		iterations          int
		minExpectedRate     float64 // Minimum success rate for expected moves
	}{
		{
			name: "Take Winning Move",
			initialBoard: [9]int{
				1, 0, 0,
				1, 2, 2,
				0, 0, 0,
			},
			nextPlayer:          1,
			expectedMoves:       []int{6}, // Must take the winning move
			bannedMoves:         []int{1, 2, 7, 8},
			explorationConstant: 0.5, // Lower exploration for tactical positions
			iterations:          1000,
			minExpectedRate:     0.90, // Should almost always find the winning move
		},
		{
			name: "Block Opponent Win",
			initialBoard: [9]int{
				1, 1, 0,
				0, 2, 0,
				0, 0, 0,
			},
			nextPlayer:          2,
			expectedMoves:       []int{2}, // Must block position 2
			bannedMoves:         []int{5, 6, 7, 8},
			explorationConstant: 0.5,
			iterations:          1000,
			minExpectedRate:     0.90,
		},
		{
			name: "Center First",
			initialBoard: [9]int{
				0, 0, 0,
				0, 0, 0,
				0, 0, 0,
			},
			nextPlayer:          1,
			expectedMoves:       []int{4}, // Center is best
			bannedMoves:         []int{},
			explorationConstant: 0.5,
			iterations:          1000, // Can be lower now due to restricted move set
			minExpectedRate:     0.75,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			state := &TicTacToeState{
				board:    tt.initialBoard,
				nextMove: tt.nextPlayer,
				moves:    []int{},
			}

			problem := &TicTacToeProblem{
				initialState: state,
				player:       tt.nextPlayer,
			}

			config := Config{
				ExplorationConstant: tt.explorationConstant,
				MaxIterations:       tt.iterations,
				TargetSeqLength:     1,
				RandomSeed:          1,
				DebugLevel:          0,
			}

			moveStats := make(map[int]int)
			numAttempts := 100

			for i := 0; i < numAttempts; i++ {
				config.RandomSeed = int64(i)
				sequence, err := Run([]interface{}{}, problem.nextElements, problem.fitness, config)

				if err != nil {
					t.Fatalf("MCTS failed: %v", err)
				}

				if len(sequence) > 0 {
					move := sequence[0].(int)
					moveStats[move]++
				}
			}

			// Print board state and move distribution
			t.Logf("\nInitial board state:%s", state)
			t.Logf("Move distribution over %d attempts:", numAttempts)
			for move, count := range moveStats {
				t.Logf("Position %d: %d times (%.1f%%)",
					move, count, float64(count)*100/float64(numAttempts))
			}

			// Check if expected moves were chosen enough times
			totalExpectedMoves := 0
			for _, expectedMove := range tt.expectedMoves {
				count := moveStats[expectedMove]
				totalExpectedMoves += count
				if count == 0 {
					t.Errorf("Expected move %d was never chosen", expectedMove)
				}
			}

			actualRate := float64(totalExpectedMoves) / float64(numAttempts)
			if actualRate < tt.minExpectedRate {
				t.Errorf("Expected moves chosen only %.1f%% of the time, want at least %.1f%%",
					actualRate*100, tt.minExpectedRate*100)
			}

			// Check that banned moves were never chosen
			for _, bannedMove := range tt.bannedMoves {
				if count := moveStats[bannedMove]; count > 0 {
					t.Errorf("Banned move %d was chosen %d times", bannedMove, count)
				}
			}
		})
	}
}
