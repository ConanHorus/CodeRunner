package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// ItemKey opens the locked doors of the boss room.
	ItemKey ItemKind = iota

	// ItemBow arms the player with WeaponBow.
	ItemBow

	// ItemHeart puts heartHealing health back on the player. It stays on the
	// floor if the player is already at full health.
	ItemHeart
)

const (
	// heartHealing is how much health a heart puts back.
	heartHealing = 1

	// itemBobHeight is how far, in pixels, an item drifts up and down to catch
	// the eye, and itemBobSpeed is how fast, in radians per tick, it does so.
	itemBobHeight = float32(1.5)
	itemBobSpeed  = 0.08
)

var (
	bowStringColor  = color.RGBA{R: 0xCD, G: 0xD6, B: 0xF4, A: 0xFF}
	bowWoodColor    = color.RGBA{R: 0xFA, G: 0xB3, B: 0x87, A: 0xFF}
	heartColor      = color.RGBA{R: 0xF3, G: 0x8B, B: 0xA8, A: 0xFF}
	heartShineColor = color.RGBA{R: 0xF5, G: 0xE0, B: 0xDC, A: 0xFF}
	keyColor        = color.RGBA{R: 0xF9, G: 0xE2, B: 0xAF, A: 0xFF}

	// heartRows is the pixel art for a heart, one filled run per row as left
	// offset, top offset and width within a GridSize square.
	heartRows = [][3]float32{
		{3, 3, 3}, {10, 3, 3},
		{2, 4, 5}, {9, 4, 5},
		{2, 5, 12},
		{2, 6, 12},
		{2, 7, 12},
		{3, 8, 10},
		{4, 9, 8},
		{5, 10, 6},
		{6, 11, 4},
		{7, 12, 2},
	}
)

// Item is something lying on the dungeon floor that the player picks up by
// stepping onto it. Items never move or act on their own; they only draw
// themselves, bobbing gently so they stand out from the floor.
type Item struct {
	kind     ItemKind
	position Vector
	world    *World
}

// ItemKind is which item is lying on a tile.
type ItemKind uint8

// NewItem creates an item lying on a tile.
//
// Parameters:
//   - world: the world the item lies in, read for the bob animation.
//   - kind: which item it is.
//   - position: the tile it lies on.
//
// Returns:
//   - item: a ready-to-draw Item.
func NewItem(world *World, kind ItemKind, position Vector) (item *Item) {
	return &Item{kind: kind, position: position, world: world}
}

// Draw renders the item onto screen, bobbing with the world clock.
//
// Parameters:
//   - screen: the destination image for this frame.
func (this *Item) Draw(screen *ebiten.Image) {
	left := float32(this.position.X * GridSize)
	top := float32(this.position.Y*GridSize) + this.bob()

	switch this.kind {
	case ItemKey:
		drawKeyGlyph(screen, left, top)
	case ItemBow:
		drawBowGlyph(screen, left, top)
	case ItemHeart:
		drawHeartGlyph(screen, left, top)
	}
}

// Kind reports which item this is.
//
// Returns:
//   - kind: the item kind.
func (this *Item) Kind() (kind ItemKind) {
	return this.kind
}

// Position reports where the item lies.
//
// Returns:
//   - position: the item position, in tiles.
func (this *Item) Position() (position Vector) {
	return this.position
}

// SetPosition moves the item without checking the destination.
//
// Parameters:
//   - position: the new item position, in tiles.
func (this *Item) SetPosition(position Vector) {
	this.position = position
}

// Update does nothing: items lie still until picked up.
func (this *Item) Update() {
}

// bob reports how far the item is drawn above or below its tile this frame.
//
// Returns:
//   - offset: the vertical offset, in pixels.
func (this *Item) bob() (offset float32) {
	return float32(math.Sin(float64(this.world.Ticks())*itemBobSpeed)) * itemBobHeight
}

// Name reports the label announcements and the heads up display use for this
// item.
//
// Returns:
//   - name: the human readable item name.
func (this ItemKind) Name() (name string) {
	switch this {
	case ItemKey:
		return "Key"
	case ItemBow:
		return "Bow"
	case ItemHeart:
		return "Heart"
	default:
		return "Unknown"
	}
}

// Weapon reports the weapon picking up this item arms the player with.
//
// Returns:
//   - weapon: the weapon, or WeaponNone for items that are not weapons.
func (this ItemKind) Weapon() (weapon Weapon) {
	if this == ItemBow {
		return WeaponBow
	}

	return WeaponNone
}

// drawBowGlyph draws a bow, strung and standing on end, in a GridSize square.
//
// Parameters:
//   - screen: the destination image.
//   - left: the square's left edge, in pixels.
//   - top: the square's top edge, in pixels.
func drawBowGlyph(screen *ebiten.Image, left float32, top float32) {
	vector.StrokeLine(screen, left+10, top+2, left+5, top+8, 2, bowWoodColor, true)
	vector.StrokeLine(screen, left+5, top+8, left+10, top+14, 2, bowWoodColor, true)
	vector.StrokeLine(screen, left+10, top+2, left+10, top+14, 1, bowStringColor, true)
}

// drawHeartGlyph draws a pixel art heart, with a shine, in a GridSize square.
//
// Parameters:
//   - screen: the destination image.
//   - left: the square's left edge, in pixels.
//   - top: the square's top edge, in pixels.
func drawHeartGlyph(screen *ebiten.Image, left float32, top float32) {
	for _, row := range heartRows {
		vector.DrawFilledRect(screen, left+row[0], top+row[1], row[2], 1, heartColor, false)
	}

	vector.DrawFilledRect(screen, left+4, top+5, 2, 1, heartShineColor, false)
}

// drawKeyGlyph draws a key, bow to the left and teeth to the right, in a
// GridSize square. The heads up display uses it too, to show the key is held.
//
// Parameters:
//   - screen: the destination image.
//   - left: the square's left edge, in pixels.
//   - top: the square's top edge, in pixels.
func drawKeyGlyph(screen *ebiten.Image, left float32, top float32) {
	vector.StrokeCircle(screen, left+5, top+8, 3, 2, keyColor, true)
	vector.DrawFilledRect(screen, left+8, top+7, 7, 2, keyColor, false)
	vector.DrawFilledRect(screen, left+11, top+9, 2, 2, keyColor, false)
	vector.DrawFilledRect(screen, left+14, top+9, 1, 3, keyColor, false)
}
