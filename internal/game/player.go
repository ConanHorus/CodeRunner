package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var playerSpeed = 4

type Player struct {
	position Vector
}

func (this *Player) Position() Vector {
	return this.position
}

func (this *Player) SetPosition(position Vector) {
	this.position = position
}

func (this *Player) Update() {
	this.movePlayer()
}

func (this *Player) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(
		screen,
		float32(this.position.X),
		float32(this.position.Y),
		GridSize,
		GridSize,
		playerColor,
		true)
}

func (this *Player) movePlayer() {
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		this.position.X -= playerSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		this.position.X += playerSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		this.position.Y -= playerSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		this.position.Y += playerSpeed
	}

	this.position.X = Clamp(this.position.X, 0, ScreenWidth-GridSize)
	this.position.Y = Clamp(this.position.Y, 0, ScreenHeight-GridSize)
}
