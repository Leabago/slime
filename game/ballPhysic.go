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
	bounceFactor float64
}

// var ballPhysicA = BallPhysic{
// 	state:        0,
// 	gravity:      0.98,
// 	friction:     0.98,
// 	radius:       20.0,
// 	speedRun:     1.0,
// 	jump:         Vector{0.0, -1.0},
// 	jumpForce:    13.0,
// 	scrambleWall: Vector{0.0, -1.0},
// }

// var ballPhysicA = BallPhysic{
// 	state:        0,
// 	gravity:      0.8,
// 	friction:     0.98,
// 	radius:       25.0,
// 	speedRun:     1.0,
// 	jump:         Vector{0.0, -0.8},
// 	jumpForce:    10.0,
// 	scrambleWall: Vector{0.0, -0.0},
// 	bounceFactor: 0.6, // 0 = no bounce, 1 = perfect bounce
// }

// noraml
// var ballPhysicA = BallPhysic{
// 	state:        0,
// 	gravity:      1,
// 	friction:     0.98,
// 	radius:       30.0,
// 	speedRun:     1.0,
// 	jump:         Vector{0.0, -0.8},
// 	jumpForce:    10.0,
// 	scrambleWall: Vector{0.0, -0.0},
// 	bounceFactor: 1, // 0 = no bounce, 1 = perfect bounce
// }

// test
var ballPhysicA = BallPhysic{
	state:        0,
	gravity:      0.95,
	friction:     0.9,
	radius:       30.0,
	speedRun:     2.0,
	jump:         Vector{0.0, -3},
	jumpForce:    10.0,
	scrambleWall: Vector{0.0, 0.0},
	bounceFactor: 0.0, // 0 = no bounce, 1 = perfect bounce
}

//  normal
// var ballPhysicB = BallPhysic{
// 	state:        1,
// 	gravity:      0.5,
// 	friction:     0.1,
// 	radius:       45,
// 	speedRun:     2,
// 	jump:         Vector{0.0, 0.0},
// 	jumpForce:    10,
// 	scrambleWall: Vector{0.0, -0.5},
// 	bounceFactor: 0.0, // 0 = no bounce, 1 = perfect bounce
// }

// test
var ballPhysicB = BallPhysic{
	state:        1,
	gravity:      1,
	friction:     0,
	radius:       45,
	speedRun:     2,
	jump:         Vector{0.0, -0.7},
	jumpForce:    20,
	scrambleWall: Vector{0.0, -3},
	bounceFactor: 0.0, // 0 = no bounce, 1 = perfect bounce
}
