package cpu

import (
	"fmt"

	"fc-emu/bus"
)

// MARK: CPUの定義
type CPU struct {
	registers registers
	bus       bus.Bus

	instructionSet instructionSet
}

// MARK: CPUのコンストラクタ
func NewCPU() *CPU {
	cpu := &CPU{
		registers: registers{
			A:  0x00,
			X:  0x00,
			Y:  0x00,
			SP: 0xFD,
			PC: 0x0000,
			P:  NewStatusRegister(),
		},
		bus: bus.NewBus(),
	}
	cpu.instructionSet = generateInstructionSet(cpu)

	return cpu
}

// MARK: N/Zフラグの更新メソッド
func (c *CPU) updateNZFlags(result uint8) {
	// Nフラグの更新
	if (result >> 7) != 0 {
		c.registers.P.Negative = true
	} else {
		c.registers.P.Negative = false
	}

	// Zフラグの更新
	if result == 0 {
		c.registers.P.Zero = true
	} else {
		c.registers.P.Zero = false
	}
}

// MARK: 実効アドレス算出メソッド
func (c *CPU) calcOperandAddress(mode AddressingMode) uint16 {
	switch mode {
	case Immediate:
		return c.registers.PC + 1
	case ZeroPage:
		return uint16(c.bus.ReadByteFrom(c.registers.PC + 1))
	case ZeroPageXIndexed:
		base := c.bus.ReadByteFrom(c.registers.PC + 1)
		return uint16(base + c.registers.X)
	case ZeroPageYIndexed:
		base := c.bus.ReadByteFrom(c.registers.PC + 1)
		return uint16(base + c.registers.Y)
	case Absolute:
		return c.bus.ReadWordFrom(c.registers.PC + 1)
	case AbsoluteXIndexed:
		base := c.bus.ReadWordFrom(c.registers.PC + 1)
		return base + uint16(c.registers.X)
	case AbsoluteYIndexed:
		base := c.bus.ReadWordFrom(c.registers.PC + 1)
		return base + uint16(c.registers.Y)
	case Relative:
		offset := int8(c.bus.ReadByteFrom(c.registers.PC + 1))
		return uint16(int32(c.registers.PC) + int32(offset))
	case Indirect:
		ptr := c.bus.ReadWordFrom(c.registers.PC + 1)
		// ページ境界をまたぐ際のバグを再現
		if (ptr & 0xFF) == 0xFF {
			lower := c.bus.ReadByteFrom(ptr)
			upper := c.bus.ReadByteFrom(ptr & 0xFF00)
			return uint16(upper)<<8 | uint16(lower)
		} else {
			return c.bus.ReadWordFrom(ptr)
		}
	case IndexedIndirect:
		base := c.bus.ReadByteFrom(c.registers.PC + 1)
		ptr := uint8(base + c.registers.X)
		lower := c.bus.ReadByteFrom(uint16(ptr))
		upper := c.bus.ReadByteFrom(uint16(ptr+1) & 0xFF)
		return uint16(upper)<<8 | uint16(lower)
	case IndirectIndexed:
		ptrBase := c.bus.ReadByteFrom(c.registers.PC + 1)
		ptr := uint8(ptrBase)

		lower := c.bus.ReadByteFrom(uint16(ptr))
		upper := c.bus.ReadByteFrom(uint16(ptr+1) & 0xFF)
		base := uint16(upper)<<8 | uint16(lower)
		return base + uint16(c.registers.Y)
	case Implied, Accumulator:
		fallthrough
	default:
		return 0x0000
	}
}

// MARK: AND命令の実装
func (c *CPU) and(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A &= value
	c.updateNZFlags(c.registers.A)
}

// MARK: LDA命令の実装
func (c *CPU) lda(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A = value
	c.updateNZFlags(c.registers.A)
}

// MARK: uint8の配列から実行
func (c *CPU) RunWithByteArray(program []uint8) {
	// Busに仮のプログラムをセット
	c.bus.Program = program

	for {
		// 命令のフェッチ
		opcode := c.bus.ReadByteFrom(c.registers.PC)

		if opcode == 0x00 {
			return
		}

		// 命令のデコード
		instruction := c.instructionSet[opcode]

		// 命令の実行
		instruction.Handler(instruction.AddressingMode)

		fmt.Printf(
			"%04X: [%s] 0x%02X, %v\n",
			c.registers.PC,
			instruction.Mnemonic,
			instruction.Opcode,
			c.registers,
		)

		// 命令長の分プログラムカウンタを進める
		c.registers.PC += uint16(instruction.Bytes)
	}
}
