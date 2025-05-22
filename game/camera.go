package game

// Camera struct
type Camera struct {
	X, Y          float64
	Width, Height float64
}

func (c *Camera) Update(playerX, playerY float64) {
	targetX := playerX - c.Width/2
	targetY := playerY - c.Height/2

	// Smooth camera movement
	c.X += (targetX - c.X) * 0.1
	c.Y += (targetY - c.Y) * 0.1

	// Keep camera within reasonable bounds
	if c.X < 0 {
		c.X = 0
	}
	// if c.Y < 0 {
	// 	c.Y = 0
	// }
}
