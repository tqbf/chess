package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/nlopes/slack"
	"github.com/tqbf/chess"
)

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

// Game describes a current running game
type Game struct {
	// Board is the current chess board
	Board chess.Board

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

	Allowed bool

	// Moves is the history of all previous moves
	Moves []string

	// Boards is the history of all previous boards
	Previous []chess.Board
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
			log.Printf("can't find channel with id %s", ev.Channel)
			return nil
		}
	}

	if us := inf.GetUserByID(ev.User); us != nil {
		user = us.Name
	} else {
		user, ok = Users[ev.User]
		if !ok {
			log.Printf("can't find user with id %s", ev.User)
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
func (ctx *Context) DrawBoard(board chess.Board, format string, args ...interface{}) {
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
			Board: chess.StartingBoard.Normalize(),
		}
		games[ctx.Channel] = game
	}

	if !game.Allowed && !match("yo.*chessbot", ctx.Text) && !match("chess.*ok.*here", ctx.Text) && !match("help.*me.*chessbot", ctx.Text) {
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

	case match("([A-Ha-h][1-8])\\s?([A-Ha-h][1-8])", ctx.Text) || chess.AlgebraicRx.MatchString(ctx.Text):
		if game.Winner != "" {
			ctx.Post("%s has already won this game. Reset the game to make moves.", game.Winner)
			return
		}

		var start, end string
		var err error

		if chess.AlgebraicRx.MatchString(ctx.Text) {
			white := false
			if game.White == ctx.User && game.PlayingWhite {
				white = true
			} else if game.Black != ctx.User {
				// don't bother
				return
			}

			if start, end, err = game.Board.Algebraic(ctx.Text, white); err != nil {
				ctx.Post("I can't understand that move: %s", err)
				return
			}
		} else {
			tox := matches("([A-Ha-h][1-8])\\s?([A-Ha-h][1-8])", ctx.Text)
			start = strings.ToUpper(tox[1])
			end = strings.ToUpper(tox[2])
		}

		alg, _ := game.Board.CoordsToAlgebraic(start, end)

		move := func() error {
			board, err := game.Board.Move(start, end)
			if err != nil {
				return err
			}
			game.Previous = append(game.Previous, game.Board)
			if alg != "" {
				game.Moves = append(game.Moves, alg)
			} else {
				game.Moves = append(game.Moves, fmt.Sprintf("%s %s", start, end))
			}
			game.Board = board
			return nil
		}

		if ctx.User != game.White && ctx.User != game.Black {
			return
		} else if ctx.User == game.White && game.PlayingWhite {
			if err := move(); err != nil {
				ctx.Post("That's not a valid move: %s", err)
				return
			}

			game.WhiteElapsed += time.Since(game.TickFrom)
			game.PlayingWhite = false
			game.TickFrom = time.Now()

			ctx.DrawBoard(game.Board, "White (%s) moves %s(%s -> %s), white has taken %s total", game.White, alg, start, end, game.WhiteElapsed)

		} else if ctx.User == game.Black && !game.PlayingWhite {
			if err := move(); err != nil {
				ctx.Post("That's not a valid move: %s", err)
				return
			}

			game.BlackElapsed += time.Since(game.TickFrom)
			game.TickFrom = time.Now()
			game.PlayingWhite = true

			ctx.DrawBoard(game.Board, "Black (%s) moves %s(%s -> %s), black has taken %s total", game.Black, alg, start, end, game.WhiteElapsed)
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

	case match("chess.*history", ctx.Text):
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

	case match("chess.*board", ctx.Text):
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

	case match("knock.*out.*([A-Ha-h][1-9])", ctx.Text):
		tox := matches("knock.*out.*([A-Ha-h][1-9])", ctx.Text)
		start := strings.ToUpper(tox[1])

		pos, _ := game.Board.Position(start)
		game.Board = game.Board.Replace(rune('_'), pos)
		ctx.Post("Removed piece (if any) at %s.", tox[1])

	case match("move.*game.*to.*(.*?)", ctx.Text):
		tox := matches("move.*game.*to.*#?(.*?)", ctx.Text)

		if _, ok := Channels[tox[1]]; !ok {
			ctx.Post("I don't know a channel called '%s', so I can't move the game there.", tox[1])
			return
		}

		games[tox[1]] = game
		games[ctx.Channel] = &Game{
			Board: chess.StartingBoard.Normalize(),
		}

		ctx.Post("Ok, I've moved this game to #%s and reset the game in this channel.", tox[1])

	case match("yo.*chess.*bot", ctx.Text):
		fallthrough
	case match("what.*up.*chess", ctx.Text):
		msg := &bytes.Buffer{}
		fmt.Fprintf(msg, "I'm OK.\n")

		if !game.Allowed {
			fmt.Fprintf(msg, "Chess commands are NOT allowed here; say 'chess is ok here' to allow them\n")
		}

		if game.Winner != "" {
			fmt.Fprintf(msg, "The current game is over, and *%s* won it. Say 'reset game' to start a new one\n", game.Winner)
		} else if len(game.Moves) > 0 {
			fmt.Fprintf(msg, "We're %d moves into the current game.\n", len(game.Moves))
		}

		if game.White != "" {
			fmt.Fprintf(msg, "*%s* is playing white", game.White)
			if game.WhiteOk {
				fmt.Fprintf(msg, ", and is ready; %s elapsed", game.WhiteElapsed)
			}
			fmt.Fprintf(msg, "\n")
		} else {
			fmt.Fprintf(msg, "Nobody has claimed white.\n")
		}

		if game.Black != "" {
			fmt.Fprintf(msg, "*%s* is playing black", game.Black)
			if game.BlackOk {
				fmt.Fprintf(msg, ", and is ready; %s elapsed", game.BlackElapsed)
			}
			fmt.Fprintf(msg, "\n")
		} else {
			fmt.Fprintf(msg, "Nobody has claimed black.\n")
		}

		fmt.Fprintf(msg, "Please do not pentest chessbot3000.\n")

		ctx.Post(msg.String())

	case match("definitely.*reset", ctx.Text):
		ctx.Post("OK. I've reset the game. New players should claim spots and start.")
		game = &Game{
			Board: chess.StartingBoard.Normalize(),
		}
		games[ctx.Channel] = game

	case match("reset.*game", ctx.Text):
		ctx.Post("Are you sure? Say 'definitely reset' if you are.")

	case match("chess.*ok.*here", ctx.Text):
		game.Allowed = true
		ctx.Post("Ok. I'll allow chess games here.")

	case match("no.*chess.*here", ctx.Text):
		game.Allowed = false
		ctx.Post("Ok. I won't respond to chess events on this channel.")

	case match("help.*me.*chessbot", ctx.Text):
		fallthrough
	case match("chess help", ctx.Text):
		ctx.Post(`Here's what I know how to do (all commands case-insensitive):
_chess is ok here, thank you_: Allow chess events on this channel
_claim_ _white_ (or _black_): Take a side
_start_: Game starts once both players say this
_A1 B2_ or _a1b2_: Make a move. *Only minimal validation is done.*
_take back_: Take a move back
_knock out D4_: Take the pawn at D4 en passant (or, you know, any other square)
_history_: See all previous moves
_board_ <num>: Display earlier board #<num>
_board_: Display the current board
_reset game_: Start over
_i resign_: Resign the game
_black_ (or _white_) _wins_: Declare a winner
_keep playing_: Un-declare a winner
_move game to #foo_: Move the game to another channel. Stop annoying people.
_what's up chessbot_: Current status
_no chess here_: Don't listen for chess stuff on this channel

*Please do not pentest chessbot3000.*
`)
	}
}
