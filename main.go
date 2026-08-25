package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/ConanHorus/CodeRunner/internal/game"
	"github.com/ConanHorus/CodeRunner/internal/tools"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	source, err := tools.PromptForFileContents()
	if errors.Is(err, tools.ErrCanceled) {
		fmt.Println("Canceled.")
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("loaded %d bytes of source", len(source))

	ebiten.SetWindowSize(game.ScreenWidth*2, game.ScreenHeight*2)
	ebiten.SetWindowTitle("CodeRunner")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	err = ebiten.RunGame(game.New(source))
	if err != nil {
		log.Fatal(err)
	}
}
