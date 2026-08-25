package main

import (
	"log"

	"github.com/ConanHorus/CodeRunner/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(game.ScreenWidth*2, game.ScreenHeight*2)
	ebiten.SetWindowTitle("CodeRunner")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	err := ebiten.RunGame(game.New())
	if err != nil {
		log.Fatal(err)
	}
}
