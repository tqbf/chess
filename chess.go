package chess

// A simple Slack chessboard bot

import (
	"fmt"
	"regexp"

	"strings"
	"unicode"
)

// Convert my ASCII chess notation to Unicode chess
var AsciiMap = map[string]string{
	"K": B_KING,
	"Q": B_QUEEN,
	"R": B_ROOK,
	"B": B_BISHOP,
	"N": B_KNIGHT,
	"P": B_PAWN,
}

// A Board is a 64-character string representing a chessboard, index 0 is A8, 1 is B8
type Board string

const (
	StartingBoard Board = `
RNBQKBNR
PPPPPPPP
________
________
________
________
pppppppp
rnbqkbnr
`
)

// Normalize converts a formatted string into one we can move pieces on; for now,
// this just strips spaces.
func (self Board) Normalize() Board {
	r := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, string(self))
	return Board(r)
}

// Position returns the index into a board at a given chessboard coordinate
// BUG(tqbf): misnamed.
func (board Board) Position(pos string) (int, error) {
	pos = strings.ToUpper(pos)

	if pos[0] < 'A' || pos[0] > 'H' {
		return -1, fmt.Errorf("bad column '%s'", string(pos[0]))
	}

	if pos[1] < '1' || pos[1] > '8' {
		return -1, fmt.Errorf("bad row '%s'", string(pos[1]))
	}

	r := int(pos[1] - 48)
	p := ((8 - r) * 8)
	p += (int(pos[0] - 'A'))

	return p, nil
}

func (board Board) Coord(pos string) (rune, int, error) {
	pos = strings.ToUpper(pos)

	if pos[0] < 'A' || pos[0] > 'H' {
		return ' ', 0, fmt.Errorf("bad column '%s'", string(pos[0]))
	}

	if pos[1] < '1' || pos[1] > '8' {
		return ' ', 0, fmt.Errorf("bad row '%s'", string(pos[1]))
	}

	return rune(pos[0]), int(pos[1] - 48), nil
}

// PieceAt returns the ASCII code of the piece at the chessboard coordinate,
// '_' if no piece, and an error if the coordinate can't be parsed
func (board Board) PieceAt(pos string) (string, error) {
	p, err := board.Position(pos)
	if err == nil {
		return string(board[p]), nil
	}
	return "", err
}

func (board Board) Replace(r rune, i int) Board {
	out := []rune(string(board))
	out[i] = r
	return Board(out)
}

type coord struct {
	row, col int
}

func coordsFromIndex(pos int) (row int, col int) {
	row = 8 - (pos / 8)
	col = pos % 8
	return
}

func indexFromCoords(row, col int) int {
	return ((8 - row) * 8) + col
}

func (board Board) validMoves(pos int) (valid []coord) {

	or, oc := coordsFromIndex(pos)

	piece := board[pos]

	empty := func(r, c int) bool {
		o := board[indexFromCoords(r, c)]
		if o == '_' {
			return true
		}
		return false
	}

	opponentAt := func(r, c int) bool {
		o := board[indexFromCoords(r, c)]
		if empty(r, c) {
			return false
		}

		if piece < 'Z' {
			// black

			if o >= 'a' {
				return true
			}
		} else {
			// white

			if o < 'Z' {
				return true
			}
		}

		return false
	}

	yep := func(r, c int) {
		valid = append(valid, coord{r, c})
	}

	cruise := func(r, c int) bool {
		if empty(r, c) {
			yep(r, c)
			return true
		}

		if opponentAt(r, c) {
			yep(r, c)
		}

		return false
	}

	switch piece {
	case 'P':
		if or != 1 {
			if empty(or-1, oc) {
				yep(or-1, oc)
			}

			if oc != 0 && opponentAt(or-1, oc-1) {
				yep(or-1, oc-1)
			}

			if oc != 7 && opponentAt(or-1, oc+1) {
				yep(or-1, oc+1)
			}

			if or == 7 && empty(or-2, oc) {
				yep(or-2, oc)
			}

			if or == 4 && oc != 0 && board[indexFromCoords(or, oc-1)] == 'p' {
				yep(or-1, oc-1)
			}

			if or == 4 && oc != 7 && board[indexFromCoords(or, oc+1)] == 'p' {
				yep(or-1, oc+1)
			}
		}

	case 'p':
		if or != 8 {
			if empty(or+1, oc) {
				yep(or+1, oc)
			}

			if oc != 0 && opponentAt(or+1, oc-1) {
				yep(or+1, oc-1)
			}

			if oc != 7 && opponentAt(or+1, oc+1) {
				yep(or+1, oc+1)
			}

			if or == 2 && empty(or+2, oc) {
				yep(or+2, oc)
			}

			if or == 5 && oc != 0 && board[indexFromCoords(or, oc-1)] == 'P' {
				yep(or+1, oc-1)
			}

			if or == 5 && oc != 7 && board[indexFromCoords(or, oc+1)] == 'P' {
				yep(or+1, oc+1)
			}
		}

	case 'R', 'r':
		for i := (or + 1); i <= 8 && cruise(i, oc); i++ {
		}

		for i := (or - 1); i >= 1 && cruise(i, oc); i-- {
		}

		for i := (oc + 1); i <= 7 && cruise(or, i); i++ {
		}

		for i := (oc - 1); i >= 0 && cruise(or, i); i-- {
		}

	case 'B', 'b':
		for tr, tc := or+1, oc+1; tr <= 8 && tc <= 7 && cruise(tr, tc); tr, tc = tr+1, tc+1 {
		}
		for tr, tc := or+1, oc-1; tr <= 8 && tc >= 0 && cruise(tr, tc); tr, tc = tr+1, tc-1 {
		}
		for tr, tc := or-1, oc+1; tr >= 1 && tc <= 7 && cruise(tr, tc); tr, tc = tr-1, tc+1 {
		}
		for tr, tc := or-1, oc-1; tr >= 1 && tc >= 0 && cruise(tr, tc); tr, tc = tr-1, tc-1 {
		}

	case 'N', 'n':
		tr, tc := or, oc

		if (tr+2 <= 8 && tc+1 <= 7) && (empty(tr+2, tc+1) || opponentAt(tr+2, tc+1)) {
			yep(tr+2, tc+1)
		}

		if (tr+2 <= 8 && tc-1 >= 0) && (empty(tr+2, tc-1) || opponentAt(tr+2, tc-1)) {
			yep(tr+2, tc-1)
		}

		if (tr-2 >= 1 && tc+1 <= 7) && (empty(tr-2, tc+1) || opponentAt(tr-2, tc+1)) {
			yep(tr-2, tc+1)
		}

		if (tr-2 >= 1 && tc-1 >= 0) && (empty(tr-2, tc-1) || opponentAt(tr-2, tc-1)) {
			yep(tr-2, tc-1)
		}

		if (tr+1 <= 8 && tc+2 <= 7) && (empty(tr+1, tc+2) || opponentAt(tr+1, tc+2)) {
			yep(tr+1, tc+2)
		}

		if (tr+1 <= 8 && tc-2 >= 0) && (empty(tr+1, tc-2) || opponentAt(tr+1, tc-2)) {
			yep(tr+1, tc-2)
		}

		if (tr-1 >= 1 && tc+2 <= 7) && (empty(tr-1, tc+2) || opponentAt(tr-1, tc+2)) {
			yep(tr-1, tc+2)
		}

		if (tr-1 >= 1 && tc-2 >= 0) && (empty(tr-1, tc-2) || opponentAt(tr-1, tc-2)) {
			yep(tr-1, tc-2)
		}

	case 'K', 'k':
		tr, tc := or, oc

		if (tr-1 >= 1 && tc-1 >= 0) && (empty(tr-1, tc-1) || opponentAt(tr-1, tc-1)) {
			yep(tr-1, tc-1)
		}
		if (tr-1 >= 1 && tc-0 >= 0) && (empty(tr-1, tc-0) || opponentAt(tr-1, tc-0)) {
			yep(tr-1, tc-0)
		}
		if (tr-0 >= 1 && tc-1 >= 0) && (empty(tr-0, tc-1) || opponentAt(tr-0, tc-1)) {
			yep(tr-0, tc-1)
		}
		if (tr+1 <= 8 && tc+1 <= 7) && (empty(tr+1, tc+1) || opponentAt(tr+1, tc+1)) {
			yep(tr+1, tc+1)
		}
		if (tr+1 <= 8 && tc+0 <= 7) && (empty(tr+1, tc+0) || opponentAt(tr+1, tc+0)) {
			yep(tr+1, tc+0)
		}
		if (tr+0 <= 8 && tc+1 <= 7) && (empty(tr+0, tc+1) || opponentAt(tr+0, tc+1)) {
			yep(tr+0, tc+1)
		}
		if (tr+1 <= 8 && tc-1 >= 0) && (empty(tr+1, tc-1) || opponentAt(tr+1, tc-1)) {
			yep(tr+1, tc-1)
		}
		if (tr-1 >= 1 && tc+1 <= 7) && (empty(tr-1, tc+1) || opponentAt(tr-1, tc+1)) {
			yep(tr-1, tc+1)
		}

	case 'Q', 'q':
		for i := (or + 1); i <= 8 && cruise(i, oc); i++ {
		}

		for i := (or - 1); i >= 1 && cruise(i, oc); i-- {
		}

		for i := (oc + 1); i <= 7 && cruise(or, i); i++ {
		}

		for i := (oc - 1); i >= 0 && cruise(or, i); i-- {
		}

		for tr, tc := or+1, oc+1; tr <= 8 && tc <= 7 && cruise(tr, tc); tr, tc = tr+1, tc+1 {
		}
		for tr, tc := or+1, oc-1; tr <= 8 && tc >= 0 && cruise(tr, tc); tr, tc = tr+1, tc-1 {
		}
		for tr, tc := or-1, oc+1; tr >= 1 && tc <= 7 && cruise(tr, tc); tr, tc = tr-1, tc+1 {
		}
		for tr, tc := or-1, oc-1; tr >= 1 && tc >= 0 && cruise(tr, tc); tr, tc = tr-1, tc-1 {
		}

	default:
		break
	}

	return
}

func (board Board) All(piece rune) (ret []int) {
	for i, p := range board {
		if p == piece {
			ret = append(ret, i)
		}
	}

	return
}

var AlgebraicRx = regexp.MustCompile("([PBNRQK])([a-h])?([1-8])?(x)?([a-h][1-8])")

func (board Board) CoordsToAlgebraic(srcs, dsts string) (string, error) {
	src, err := board.Position(srcs)
	if err != nil {
		return "", err
	}

	dst, err := board.Position(dsts)
	if err != nil {
		return "", err
	}

	if board[src] == '_' {
		return "", fmt.Errorf("no piece at %s", srcs)
	}

	x := ""
	if board[dst] != '_' {
		x = "x"
	}

	white := false
	if board[src] >= 'a' {
		white = true
	}

	piece := board[src]
	if piece >= 'a' {
		piece -= 32
	}

	try := fmt.Sprintf("%c%s%s", piece, x, strings.ToLower(dsts))
	if _, _, err := board.Algebraic(try, white); err == nil {
		return try, nil
	}

	r, c := coordsFromIndex(src)
	cs := 'a' + c

	try = fmt.Sprintf("%c%c%s%s", piece, cs, x, strings.ToLower(dsts))
	if _, _, err := board.Algebraic(try, white); err == nil {
		return try, nil
	}

	try = fmt.Sprintf("%c%c%d%s%s", piece, cs, r, x, strings.ToLower(dsts))
	if _, _, err := board.Algebraic(try, white); err == nil {
		return try, nil
	}

	return "", fmt.Errorf("can't find a valid move described by %s->%s", srcs, dsts)
}

func (board Board) Algebraic(move string, isWhite bool) (string, string, error) {
	if move == "O-O" {
		if isWhite {
			return "E1", "G1", nil
		} else {
			return "E8", "G8", nil
		}
	}

	if move == "O-O-O" {
		if isWhite {
			return "E1", "C1", nil
		} else {
			return "E8", "C8", nil
		}
	}

	matches := AlgebraicRx.FindStringSubmatch(move)
	if matches == nil {
		return "", "", fmt.Errorf("invalid notation: %s", move)
	}

	piece := matches[1][0]
	srccol := matches[2]
	srcrow := matches[3]
	dst, _ := board.Position(matches[5])
	_, _, _, _ = piece, srcrow, srccol, dst

	if isWhite {
		piece += 32
	}

	// disambiguate

	cands := []int{}
	for _, p := range board.All(rune(piece)) {
		r, c := coordsFromIndex(p)

		if srccol != "" {
			dcol := int(srccol[0] - 'a')
			if dcol != c {
				continue
			}
		}

		if srcrow != "" {
			drow := int(srcrow[0] - '0')
			if drow != r {
				continue
			}
		}

		cands = append(cands, p)
	}

	src := ""
	drow, dcol := coordsFromIndex(dst)

	// fmt.Printf("candidates: %#v\n", cands)

	for _, p := range cands {
		for _, mv := range board.validMoves(p) {
			if mv.row == drow && mv.col == dcol {
				if src != "" {
					return "", "", fmt.Errorf("ambiguous move")
				}

				srow, scol := coordsFromIndex(p)

				src = fmt.Sprintf("%c%d", 'A'+scol, srow)
			}
		}
	}

	if src == "" {
		return "", "", fmt.Errorf("no matching move")
	}

	return src, matches[5], nil
}

// Move moves pieces on a board, returning the new board, or an error if
// the move is invalid. Takes A8, H1 style coordinates (we don't parse
// algebraic right now). Does only the most minimal validation. Does
// handle queen promotion.
func (board Board) Move(starts, stops string) (Board, error) {
	starts = strings.ToUpper(starts)
	stops = strings.ToUpper(stops)

	start, err := board.Position(starts)
	if err != nil {
		return board, err
	}

	stop, err := board.Position(stops)
	if err != nil {
		return board, err
	}

	piece := board[start]

	if piece == '_' {
		return board, fmt.Errorf("no piece at %s", starts)
	}

	if piece == 'k' && starts == "E1" && (stops == "G1" || stops == "C1") {
		var rs, re int
		if strings.ToUpper(stops) == "G1" {
			rs, _ = board.Position("H1")
			re, _ = board.Position("F1")
		} else {
			rs, _ = board.Position("A1")
			re, _ = board.Position("D1")
		}

		board = board.
			Replace(rune('k'), stop).
			Replace(rune('r'), re).
			Replace(rune('_'), start).
			Replace(rune('_'), rs)

		return board, nil
	}

	if piece == 'K' && starts == "E8" && (stops == "G8" || stops == "C8") {
		var rs, re int
		if strings.ToUpper(stops) == "G8" {
			rs, _ = board.Position("H8")
			re, _ = board.Position("F8")
		} else {
			rs, _ = board.Position("A8")
			re, _ = board.Position("D8")
		}
		board = board.
			Replace(rune('K'), stop).
			Replace(rune('R'), re).
			Replace(rune('_'), start).
			Replace(rune('_'), rs)
		return board, nil
	}

	if piece == 'p' && stops[1] == '8' {
		piece = 'q'
	} else if piece == 'P' && stops[1] == '1' {
		piece = 'Q'
	}

	board = board.Replace(rune(piece), stop).Replace('_', start)
	return board, nil
}
