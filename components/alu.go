package components

// ALU, or the arithmetic-logical unit, performs calculations
// based on the two 16-bit inputs x and y, and the opcode composed
// of 6 separate channels:
//   zx => x = 0
//   nx => x = !x
//   zy => y = 0
//   ny => y = !y
//   f  => out = two's compliment x + y, else out = x & y
//   no => out = !out
type ALU struct {
	x, y                  Val // inputs
	zx, nx, zy, ny, f, no Val // flags

	// x preprocessing gates
	zxMux *Mux16
	nxNot *Not16
	nxMux *Mux16

	// y preprocessing gates
	zyMux *Mux16
	nyNot *Not16
	nyMux *Mux16

	// function gates
	fAdd *Add16
	fAnd *And16
	fMux *Mux16

	// postprocess gates
	noNot *Not16
	noMux *Mux16

	// zero flag gatest
	zrOr8Way1 *Or8Way
	zrOr8Way2 *Or8Way
	zrOr      *Or
	zrNot     *Not
}

func NewALU() *ALU {
	return &ALU{
		&InvalidVal{}, &InvalidVal{},
		&InvalidVal{}, &InvalidVal{}, &InvalidVal{},
		&InvalidVal{}, &InvalidVal{}, &InvalidVal{},
		NewMux16(), NewNot16(), NewMux16(),
		NewMux16(), NewNot16(), NewMux16(),
		NewAdd16(), NewAnd16(), NewMux16(),
		NewNot16(), NewMux16(),
		NewOr8Way(), NewOr8Way(), NewOr(), NewNot(),
	}
}

// Update updates the ALU's channels and returns the result of
// the computation as three different components in the following
// order:
//   1. result of the computation, a SixteenChannel,
//   2. zero flag, a SingleChannel, true if the result of the computation
//      equals to 0,
//   3. negative flag, a SingleChannel, true if the result of the computation
//      is less than 0.
//
// Inputs x and y can be updated with an UpdateOpts to TargetX and TargetY,
// respectively. The UpdateOpts value must be a SixteenChan.
//
// The following table represents the valid opcodes and their output:
//
// | zx  | nx  | zy  | ny  |  f  | no  | out |
// |-----|-----|-----|-----|-----|-----|-----|
// |  1  |  0  |  1  |  0  |  1  |  0  |  0  |
// |  1  |  1  |  1  |  1  |  1  |  1  |  1  |
// |  1  |  1  |  1  |  0  |  1  |  0  | -1  |
// |  0  |  0  |  1  |  1  |  0  |  0  |  x  |
// |  1  |  1  |  0  |  0  |  0  |  0  |  y  |
// |  0  |  0  |  1  |  1  |  0  |  1  | !x  |
// |  1  |  1  |  0  |  0  |  0  |  1  | !y  |
// |  0  |  0  |  1  |  1  |  1  |  1  | -x  |
// |  1  |  1  |  0  |  0  |  1  |  1  | -y  |
// |  0  |  1  |  1  |  1  |  1  |  1  | x+1 |
// |  1  |  1  |  0  |  1  |  1  |  1  | y+1 |
// |  0  |  0  |  1  |  1  |  1  |  0  | x-1 |
// |  1  |  1  |  0  |  0  |  1  |  0  | y-1 |
// |  0  |  0  |  0  |  0  |  1  |  0  | x+y |
// |  0  |  1  |  0  |  0  |  1  |  1  | x-y |
// |  0  |  0  |  0  |  1  |  1  |  1  | y-x |
// |  0  |  0  |  0  |  0  |  0  |  0  | x&y |
// |  0  |  1  |  0  |  1  |  0  |  1  | x|y |
//
// The column name to UpdateOpts target is the following:
//   zx = TargetZeroX
//   nx = TargetNegX
//   zy = TargetZeroY
//   ny = TargetNegY
//   f  = TargetFunc
//   no = TargetNegOut
func (alu *ALU) Update(opts ...UpdateOpts) (Val, Val, Val) {
	for _, opt := range opts {
		switch opt.target {
		case TargetX:
			alu.x = opt.val
		case TargetY:
			alu.y = opt.val
		case TargetZeroX:
			alu.zx = opt.val
		case TargetNegX:
			alu.nx = opt.val
		case TargetZeroY:
			alu.zy = opt.val
		case TargetNegY:
			alu.ny = opt.val
		case TargetFunc:
			alu.f = opt.val
		case TargetNegOut:
			alu.no = opt.val
		}
	}

	// preprocess x
	xZero := alu.zxMux.Update(
		UpdateOpts{TargetA, alu.x},
		UpdateOpts{TargetB, &SixteenChan{0}},
		UpdateOpts{TargetSel0, alu.zx},
	)

	xNeg := alu.nxNot.Update(
		UpdateOpts{TargetIn, xZero},
	)
	xPreprocessed := alu.nxMux.Update(
		UpdateOpts{TargetA, xZero},
		UpdateOpts{TargetB, xNeg},
		UpdateOpts{TargetSel0, alu.nx},
	)

	// preprocess y
	yZero := alu.zyMux.Update(
		UpdateOpts{TargetA, alu.y},
		UpdateOpts{TargetB, &SixteenChan{0}},
		UpdateOpts{TargetSel0, alu.zy},
	)

	yNeg := alu.nyNot.Update(
		UpdateOpts{TargetIn, yZero},
	)
	yPreprocessed := alu.nyMux.Update(
		UpdateOpts{TargetA, yZero},
		UpdateOpts{TargetB, yNeg},
		UpdateOpts{TargetSel0, alu.ny},
	)

	// function(x, y)
	xyAdd := alu.fAdd.Update(
		UpdateOpts{TargetA, xPreprocessed},
		UpdateOpts{TargetB, yPreprocessed},
	)
	xyAnd := alu.fAnd.Update(
		UpdateOpts{TargetA, xPreprocessed},
		UpdateOpts{TargetB, yPreprocessed},
	)
	xyF := alu.fMux.Update(
		UpdateOpts{TargetA, xyAnd},
		UpdateOpts{TargetB, xyAdd},
		UpdateOpts{TargetSel0, alu.f},
	)

	// postprocess xyF
	negXy := alu.noNot.Update(
		UpdateOpts{TargetIn, xyF},
	)
	result := alu.noMux.Update(
		UpdateOpts{TargetA, xyF},
		UpdateOpts{TargetB, negXy},
		UpdateOpts{TargetSel0, alu.no},
	)

	// set status flags
	ng := SingleChan{result.GetBoolFromUint16(15)}

	loByteOr := alu.zrOr8Way1.Update(
		UpdateOpts{TargetA, &SingleChan{result.GetBoolFromUint16(0)}},
		UpdateOpts{TargetB, &SingleChan{result.GetBoolFromUint16(1)}},
		UpdateOpts{TargetC, &SingleChan{result.GetBoolFromUint16(2)}},
		UpdateOpts{TargetD, &SingleChan{result.GetBoolFromUint16(3)}},
		UpdateOpts{TargetE, &SingleChan{result.GetBoolFromUint16(4)}},
		UpdateOpts{TargetF, &SingleChan{result.GetBoolFromUint16(5)}},
		UpdateOpts{TargetG, &SingleChan{result.GetBoolFromUint16(6)}},
		UpdateOpts{TargetH, &SingleChan{result.GetBoolFromUint16(7)}},
	)
	hiByteOr := alu.zrOr8Way2.Update(
		UpdateOpts{TargetA, &SingleChan{result.GetBoolFromUint16(8)}},
		UpdateOpts{TargetB, &SingleChan{result.GetBoolFromUint16(9)}},
		UpdateOpts{TargetC, &SingleChan{result.GetBoolFromUint16(10)}},
		UpdateOpts{TargetD, &SingleChan{result.GetBoolFromUint16(11)}},
		UpdateOpts{TargetE, &SingleChan{result.GetBoolFromUint16(12)}},
		UpdateOpts{TargetF, &SingleChan{result.GetBoolFromUint16(13)}},
		UpdateOpts{TargetG, &SingleChan{result.GetBoolFromUint16(14)}},
		UpdateOpts{TargetH, &SingleChan{result.GetBoolFromUint16(15)}},
	)
	zrFlag := alu.zrOr.Update(
		UpdateOpts{TargetA, loByteOr},
		UpdateOpts{TargetB, hiByteOr},
	)
	zr := alu.zrNot.Update(
		UpdateOpts{TargetIn, zrFlag},
	)

	return result, zr, &ng
}

const (
	TargetX Target = iota + 100
	TargetY
	TargetZeroX
	TargetNegX
	TargetZeroY
	TargetNegY
	TargetFunc
	TargetNegOut
)
