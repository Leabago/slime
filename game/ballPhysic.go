package game

type phyStateInt int

const (
	phyStateA phyStateInt = iota
	phyStateB
)

type BallPhysic struct {
	state        phyStateInt
	gravity      float64
	friction     float64
	radius       float64
	speedRun     float64
	jump         Vector
	jumpForce    float64
	scrambleWall Vector

	jumpVelFunc func(jumpVel, normal Vector) Vector
}

var ballPhysicA = BallPhysic{
	state:        0,
	gravity:      0.98,
	friction:     0.98,
	radius:       20.0,
	speedRun:     1.0,
	jump:         Vector{0.0, -1.0},
	jumpForce:    13.0,
	scrambleWall: Vector{0.0, -1.0},
}

var ballPhysicB = BallPhysic{
	state:        1,
	gravity:      0.5,
	friction:     0.1,
	radius:       40,
	speedRun:     2,
	jump:         Vector{0.0, 0.0},
	jumpForce:    10,
	scrambleWall: Vector{0.0, -0.5},
}
