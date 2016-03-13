package main

// A simple Slack chessboard bot

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"image"
	"image/color"
	"strconv"
	"strings"
	"unicode"

	"github.com/nlopes/slack"

	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

// Unicode chess pieces. There's a draw2d-compat ps interpreter that could get us
// nicer pieces, but it barfs on all the converted PS files I fed it. :(
// Note that due to a bug in draw2d, we just use the black pieces (colored white and
// black)
const (
	W_KING   = "♔"
	W_QUEEN  = "♕"
	W_ROOK   = "♖"
	W_BISHOP = "♗"
	W_KNIGHT = "♘" // misogynist 2d library panics on this character.
	W_PAWN   = "♙"
	B_KING   = "♚"
	B_QUEEN  = "♛"
	B_ROOK   = "♜"
	B_BISHOP = "♝"
	B_KNIGHT = "♞"
	B_PAWN   = "♟"
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

func rgb(r, g, b byte) color.RGBA {
	return color.RGBA{
		R: r,
		G: g,
		B: b,
		A: 255,
	}
}

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

// PieceAt returns the ASCII code of the piece at the chessboard coordinate,
// '_' if no piece, and an error if the coordinate can't be parsed
func (board Board) PieceAt(pos string) (string, error) {
	p, err := board.Position(pos)
	if err == nil {
		return string(board[p]), nil
	}
	return "", err
}

var (
	// The chessboard colors
	Dark  = rgb(101, 63, 55)
	Light = rgb(233, 172, 96)
)

// Draw draws a chessboard into a width x width square RGBA image
func (board Board) Draw(width int) image.Image {
	gc, dest := initializeDrawing(width)
	board.doDraw(gc)
	label(gc)
	return dest
}

func (board Board) doDraw(gc draw2d.GraphicContext) {
	gc.SetStrokeColor(&color.RGBA{
		A: 0,
	})

	for r := 8; r >= 1; r-- {
		for c := 'A'; c <= 'H'; c++ {
			pos := string(c) + string(byte(r+48))
			col := 7 - int('H'-c)
			yo := float64((8 - r) * 10)
			xo := float64(col * 10)
			fill := Dark

			if (r%2 == 0 && col%2 == 0) || (r%2 == 1 && col%2 == 1) {
				fill = Light
			}

			gc.SetFillColor(fill)

			draw2dkit.Rectangle(gc, (xo), (yo), (xo + 10), (yo + 10))
			gc.FillStroke()

			piece, _ := board.PieceAt(pos)
			if piece != "_" {
				gc.SetFillColor(image.Black)

				if piece[0] >= 'a' && piece[0] <= 'z' {
					gc.SetFillColor(image.White)
				}

				piece = AsciiMap[strings.ToUpper(piece)]

				gc.FillStringAt(piece, xo+1.5, (yo+10.0)-1.5)
			}
		}
	}
}

func replace(in Board, r rune, i int) Board {
	out := []rune(string(in))
	out[i] = r
	return Board(out)
}

// Move moves pieces on a board, returning the new board, or an error if
// the move is invalid. Takes A8, H1 style coordinates (we don't parse
// algebraic right now). Does only the most minimal validation. Does
// handle queen promotion.
func (board Board) Move(starts, stops string) (Board, error) {
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

	if piece == 'p' && stops[1] == '8' {
		piece = 'q'
	} else if piece == 'P' && stops[1] == '1' {
		piece = 'Q'
	}

	board = replace(board, rune(piece), stop)
	board = replace(board, '_', start)
	return board, nil
}

func initializeDrawing(width int) (draw2d.GraphicContext, image.Image) {
	dest := image.NewRGBA(image.Rect(0, 0, (width), (width)))
	gc := draw2dimg.NewGraphicContext(dest)
	draw2d.SetFontFolder(".")
	gc.SetFontData(draw2d.FontData{Name: "dejavu", Family: draw2d.FontFamilyMono})
	gc.SetFontSize(10)
	gc.Scale(float64(dest.Bounds().Max.X)/90.0, float64(dest.Bounds().Max.Y)/90.0)
	gc.Translate(10, 0)
	return gc, dest
}

func label(gc draw2d.GraphicContext) {
	gc.SetFillColor(rgb(100, 100, 100))
	gc.SetFontSize(4)

	gc.FillStringAt("8", -4, 9)
	gc.FillStringAt("7", -4, 19)
	gc.FillStringAt("6", -4, 29)
	gc.FillStringAt("5", -4, 39)
	gc.FillStringAt("4", -4, 49)
	gc.FillStringAt("3", -4, 59)
	gc.FillStringAt("2", -4, 69)
	gc.FillStringAt("1", -4, 79)

	gc.FillStringAt("A", 2, 85)
	gc.FillStringAt("B", 12, 85)
	gc.FillStringAt("C", 22, 85)
	gc.FillStringAt("D", 32, 85)
	gc.FillStringAt("E", 42, 85)
	gc.FillStringAt("F", 52, 85)
	gc.FillStringAt("G", 62, 85)
	gc.FillStringAt("H", 72, 85)
}

// Game describes a current running game
type Game struct {
	// Board is the current chess board
	Board Board

	// Black is the Slack user NAME (not ID) of the black player.
	Black string
	// White is the Slack user NAME (not ID) of the white player; can be same as black
	White string

	// PlayingWhite is true when it's white's move
	PlayingWhite bool

	// Winner is the Slack user name of the winning player; the game is over when
	// there's a winner (no draws right now)
	Winner string

	// BlackOk and WhiteOk determine whether it is OK to start the game
	BlackOk, WhiteOk bool

	// TickFrom is the timestamp of the last move
	TickFrom time.Time

	// WhiteElapsed is how much time has elapsed for white's moves
	WhiteElapsed time.Duration
	BlackElapsed time.Duration

	// Disallowed is true if the channel for this game doesn't allow chess events
	Disallowed bool

	// Moves is the history of all previous moves
	Moves []string

	// Boards is the history of all previous boards
	Previous []Board
}

var games = map[string]*Game{}

func match(rxs, message string) bool {
	return matches(rxs, message) != nil
}

func matches(rxs, message string) []string {
	rx := regexp.MustCompile("(?i)" + rxs)
	return rx.FindStringSubmatch(message)
}

// Context wraps up the state for a single incoming message, just so we can
// pass fewer arguments to functions
type Context struct {
	Channel string
	User    string
	Text    string
	API     *slack.Client
}

// BUG(tqbf): these shouldn't be global variables
var Channels = map[string]string{}
var Users = map[string]string{}

// ContextFromEvent creates a Context given the crap we get from the Slack RTM interface
func ContextFromEvent(api *slack.Client, inf *slack.Info, ev *slack.MessageEvent) *Context {
	var channel, user string
	var ok bool

	if ch := inf.GetChannelByID(ev.Channel); ch != nil {
		channel = ch.Name
	} else {
		channel, ok = Channels[ev.Channel]
		if !ok {
			return nil
		}
	}

	if us := inf.GetUserByID(ev.User); us != nil {
		user = us.Name
	} else {
		user, ok = Users[ev.User]
		if !ok {
			return nil
		}
	}

	return &Context{
		Channel: channel,
		User:    user,
		Text:    ev.Text,
		API:     api,
	}
}

// Post posts a simple text message to the channel on which a message was received
func (ctx *Context) Post(format string, args ...interface{}) {
	ctx.API.PostMessage("#"+ctx.Channel, fmt.Sprintf(format, args...), slack.PostMessageParameters{
		AsUser: true,
	})
}

// PostLink posts a text message with an image attachment to the channel on which a message was received
func (ctx *Context) PostLink(link, title, format string, args ...interface{}) {
	p := slack.PostMessageParameters{
		AsUser: true,
	}

	text := fmt.Sprintf(format, args...)

	p.Attachments = []slack.Attachment{
		{
			Title:    title,
			ImageURL: link,
		},
	}

	ctx.API.PostMessage("#"+ctx.Channel, text, p)
}

// DrawBoard posts a message with an attached chess board
func (ctx *Context) DrawBoard(board Board, format string, args ...interface{}) {
	dest := board.Draw(400)
	fn := fmt.Sprintf("/tmp/chess_boards/board%d.png", time.Now().Unix())
	draw2dimg.SaveToPngFile(fn, dest)

	url := fmt.Sprintf("http://sockpuppet.org:7777/%s", strings.Replace(fn, "/tmp/chess_boards/", "", -1))
	ctx.PostLink(url, "Game board", fmt.Sprintf(format, args...))
}

// Incoming handles incoming messages, parses commands, and replies to them
func (ctx *Context) Incoming() {
	if ctx.User == "chessbot3000" {
		return
	}

	game, ok := games[ctx.Channel]
	if !ok {
		game = &Game{
			Board: StartingBoard.Normalize(),
		}
		games[ctx.Channel] = game
	}

	if game.Disallowed && !match("chess.*ok.*here", ctx.Text) {
		return
	}

	switch {
	case match("claim.*black", ctx.Text):
		game.Black = ctx.User
		ctx.Post("Ok, the black player is now %s", ctx.User)

	case match("claim.*white", ctx.Text):
		game.White = ctx.User
		ctx.Post("Ok, the white player is now %s", ctx.User)

	case match("start", ctx.Text):
		if ctx.User == game.White {
			game.WhiteOk = true
		}
		if ctx.User == game.Black {
			game.BlackOk = true
		}
		if !game.WhiteOk && !game.BlackOk {
			ctx.Post("Both black and white players must say start")
		} else if !game.WhiteOk {
			ctx.Post("White (%s) must say start", game.White)
		} else if !game.BlackOk {
			ctx.Post("Black (%s) must say start", game.Black)
		} else {
			ctx.Post("I've started the game; white's clock is ticking.")
			game.PlayingWhite = true
			game.TickFrom = time.Now()
		}

	case match("([A-Ha-h][1-8])\\s?([A-Ha-h][1-8])", ctx.Text):
		if game.Winner != "" {
			ctx.Post("%s has already won this game. Reset the game to make moves.", game.Winner)
			return
		}

		tox := matches("([A-Ha-h][1-8])\\s?([A-Ha-h][1-8])", ctx.Text)
		start := strings.ToUpper(tox[1])
		end := strings.ToUpper(tox[2])

		move := func() error {
			board, err := game.Board.Move(start, end)
			if err != nil {
				return err
			}
			game.Previous = append(game.Previous, game.Board)
			game.Moves = append(game.Moves, fmt.Sprintf("%s %s", start, end))
			game.Board = board
			return nil
		}

		if ctx.User != game.White && ctx.User != game.Black {
			ctx.Post("You're not playing. 'claim' either white or black to play, or annoy people.")
		} else if ctx.User == game.White && game.PlayingWhite {
			if err := move(); err != nil {
				ctx.Post("That's not a valid move: %s", err)
				return
			}

			game.WhiteElapsed += time.Since(game.TickFrom)
			game.PlayingWhite = false
			game.TickFrom = time.Now()

			ctx.DrawBoard(game.Board, "White (%s) moves %s -> %s, white has taken %s total", game.White, start, end, game.WhiteElapsed)

		} else if ctx.User == game.Black && !game.PlayingWhite {
			if err := move(); err != nil {
				ctx.Post("That's not a valid move: %s", err)
				return
			}

			game.BlackElapsed += time.Since(game.TickFrom)
			game.TickFrom = time.Now()
			game.PlayingWhite = true

			ctx.DrawBoard(game.Board, "Black (%s) moves %s -> %s, black has taken %s total", game.Black, start, end, game.WhiteElapsed)
		} else {
			ctx.Post("It's not your turn.")
		}

	case match("take\\s?back", ctx.Text):
		if game.Winner != "" {
			ctx.Post("%s has already won this game. Reset the game to make moves.", game.Winner)
			return
		}
		if len(game.Previous) < 1 {
			ctx.Post("There are no moves to take back.")
			return
		}

		if ctx.User == game.White && !game.PlayingWhite {
			game.Board = game.Previous[len(game.Previous)-1]
			game.PlayingWhite = true
			ctx.DrawBoard(game.Board, "White (%s) takes back %s, white's move again", game.White, game.Moves[len(game.Moves)-1])
		} else if ctx.User == game.Black && game.PlayingWhite {
			game.Board = game.Previous[len(game.Previous)-1]
			game.PlayingWhite = false
			ctx.DrawBoard(game.Board, "Black (%s) takes back %s, black's move again", game.Black, game.Moves[len(game.Moves)-1])
		} else {
			ctx.Post("You can't take a move back.")
		}

	case match("history", ctx.Text):
		out := &bytes.Buffer{}
		for i, move := range game.Moves {
			fmt.Fprintf(out, "_%d_. *%s*\n", i, move)
		}
		ctx.Post("All moves:\n%s", out.String())

	case match("board.*([0-9]+)", ctx.Text):
		tox := matches("board.*([0-9]+)", ctx.Text)
		which, _ := strconv.Atoi(tox[1])
		if which > len(game.Previous) {
			ctx.Post("I can't fetch previous board %d", which)
		}

		ctx.DrawBoard(game.Previous[which], "Previous board #%d (type 'board' for current board)", which)

	case match("board", ctx.Text):
		if game.PlayingWhite {
			ctx.DrawBoard(game.Board, "The current board; it's white's (%s) move", game.White)
		} else {
			ctx.DrawBoard(game.Board, "The current board; it's black's (%s) move", game.Black)
		}

	case match("i\\s+resign", ctx.Text):
		if game.PlayingWhite {
			game.Winner = game.Black
		} else {
			game.Winner = game.White
		}
		ctx.Post("*%s* has won the game!", game.Winner)

	case match("keep.*playing", ctx.Text):
		game.Winner = ""
		ctx.Post("Ok. I've forgotten who won, so you can keep making moves.")

	case match("(black|white) win(s)?", ctx.Text):
		tox := matches("(black|white) wins", ctx.Text)
		if tox[1] == "black" {
			game.Winner = game.Black
		} else {
			game.Winner = game.White
		}
		ctx.Post("*%s* has won the game!", game.Winner)

	case match("definitely.*reset", ctx.Text):
		ctx.Post("OK. I've reset the game. New players should claim spots and start.")
		game = &Game{
			Board: StartingBoard.Normalize(),
		}
		games[ctx.Channel] = game

	case match("reset", ctx.Text):
		ctx.Post("Are you sure? Say 'definitely reset' if you are.")

	case match("chess.*ok.*here", ctx.Text):
		game.Disallowed = false
		ctx.Post("Ok. I'll allow chess games here.")

	case match("no.*chess.*here", ctx.Text):
		game.Disallowed = true
		ctx.Post("Ok. I won't respond to chess events on this channel.")

	case match("help", ctx.Text):
		ctx.Post(`Here's what I know how to do (all commands case-insensitive):
_claim_ _white_ (or _black_): Take a side
_start_: Game starts once both players say this
_A1 B2_ or _a1b2_: Make a move. *Only minimal validation is done.*
_take back_: Take a move back
_history_: See all previous moves
_board_ <num>: Display earlier board #<num>
_board_: Display the current board
_reset_: Start over
_i resign_: Resign the game
_black_ (or _white_) _wins_: Declare a winner
_keep playing_: Un-declare a winner
_no chess here_: Don't listen for chess stuff on this channel
_chess is ok here, thank you_: Allow chess events on this channel
`)
	}
}

func main() {
	os.Mkdir("/tmp/chess_boards", 0755)

	go func() {
		panic(http.ListenAndServe(":7777", http.FileServer(http.Dir("/tmp/chess_boards"))))
	}()

	api := slack.New(os.Getenv("BOT_TOKEN"))
	rtm := api.NewRTM()

	go rtm.ManageConnection()

	var inf *slack.Info

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ChannelCreatedEvent:
				Channels[ev.Channel.ID] = ev.Channel.Name

			case *slack.PresenceChangeEvent:
				users, _ := api.GetUsers()
				for _, user := range users {
					Users[user.ID] = user.Name
				}

			case *slack.ConnectedEvent:
				inf = ev.Info
				for _, channel := range ev.Info.Channels {
					Channels[channel.ID] = channel.Name
				}

				for _, user := range ev.Info.Users {
					Users[user.ID] = user.Name
				}

			case *slack.MessageEvent:
				if ctx := ContextFromEvent(api, inf, ev); ctx != nil {
					ctx.Incoming()
				}

			case *slack.HelloEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
			}
		}
	}
}
