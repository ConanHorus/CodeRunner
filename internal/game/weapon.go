package game

const (
	// WeaponNone is empty hands. Attacking with it does nothing.
	WeaponNone Weapon = iota

	// WeaponSword strikes the one tile the player is facing.
	WeaponSword

	// WeaponBow looses an arrow that flies the way the player is facing until
	// it hits a monster or a wall.
	WeaponBow
)

const (
	// bowCooldown and swordCooldown are how long, in seconds, the player must
	// wait between attacks with each weapon. The sword is quicker to make up
	// for having to stand next to what it hits.
	bowCooldown   = float32(0.45)
	swordCooldown = float32(0.22)
)

// Weapon is one of the arms the player can pick up and fight with.
type Weapon uint8

// Cooldown reports how long the player must wait after attacking with this
// weapon before attacking again.
//
// Returns:
//   - cooldown: the wait, in seconds. It is zero for WeaponNone.
func (this Weapon) Cooldown() (cooldown float32) {
	switch this {
	case WeaponSword:
		return swordCooldown
	case WeaponBow:
		return bowCooldown
	default:
		return 0
	}
}

// Name reports the label the heads up display shows for this weapon.
//
// Returns:
//   - name: the human readable weapon name.
func (this Weapon) Name() (name string) {
	switch this {
	case WeaponSword:
		return "Sword"
	case WeaponBow:
		return "Bow"
	default:
		return "No weapon"
	}
}
