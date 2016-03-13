package chess

import (
	"image"
	"image/color"
	"strings"

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

func rgb(r, g, b byte) color.RGBA {
	return color.RGBA{
		R: r,
		G: g,
		B: b,
		A: 255,
	}
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
