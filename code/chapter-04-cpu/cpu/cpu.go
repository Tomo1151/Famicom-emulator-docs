package cpu

import (
	"fmt"
	"os"

	"fc-emu/bus"
)

// MARK: CPUの定義
type CPU struct {
	registers      registers
	bus            bus.Bus
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

// MARK: 命令サイクルの実行メソッド
func (c *CPU) Step() {
	// 命令のフェッチ
	opcode := c.bus.ReadByteFrom(c.registers.PC)
	c.registers.PC++

	// FIXME: テスト用としてBRK命令で終了
	if opcode == 0x00 {
		os.Exit(0)
	}

	// 命令のデコード
	instruction := c.instructionSet[opcode]

	// 命令の実行
	instruction.Handler(instruction.AddressingMode)

	// PCを書き換えない命令のみ，命令長の分プログラムカウンタを進める（オペコード分 -1）
	if !instruction.Jump {
		c.registers.PC += uint16(instruction.Bytes - 1)
	}
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
		return c.registers.PC
	case ZeroPage:
		return uint16(c.bus.ReadByteFrom(c.registers.PC))
	case ZeroPageXIndexed:
		base := c.bus.ReadByteFrom(c.registers.PC)
		return uint16(base + c.registers.X)
	case ZeroPageYIndexed:
		base := c.bus.ReadByteFrom(c.registers.PC)
		return uint16(base + c.registers.Y)
	case Absolute:
		return c.bus.ReadWordFrom(c.registers.PC)
	case AbsoluteXIndexed:
		base := c.bus.ReadWordFrom(c.registers.PC)
		return base + uint16(c.registers.X)
	case AbsoluteYIndexed:
		base := c.bus.ReadWordFrom(c.registers.PC)
		return base + uint16(c.registers.Y)
	case Relative:
		offset := int8(c.bus.ReadByteFrom(c.registers.PC))
		return uint16(int32(c.registers.PC) + int32(offset))
	case Indirect:
		ptr := c.bus.ReadWordFrom(c.registers.PC)
		// ページ境界をまたぐ際のバグを再現
		if (ptr & 0xFF) == 0xFF {
			lower := c.bus.ReadByteFrom(ptr)
			upper := c.bus.ReadByteFrom(ptr & 0xFF00)
			return uint16(upper)<<8 | uint16(lower)
		} else {
			return c.bus.ReadWordFrom(ptr)
		}
	case IndexedIndirect:
		base := c.bus.ReadByteFrom(c.registers.PC)
		ptr := uint8(base + c.registers.X)
		lower := c.bus.ReadByteFrom(uint16(ptr))
		upper := c.bus.ReadByteFrom(uint16(ptr+1) & 0xFF)
		return uint16(upper)<<8 | uint16(lower)
	case IndirectIndexed:
		ptrBase := c.bus.ReadByteFrom(c.registers.PC)
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

// MARK: スタック操作
// スタック領域へのプッシュ (1バイト)
func (c *CPU) pushByte(value uint8) {
	ptr := 0x0100 | uint16(c.registers.SP)
	c.bus.WriteByteAt(ptr, value)
	c.registers.SP--
}

// スタック領域へのプッシュ (2バイト)
func (c *CPU) pushWord(value uint16) {
	ptr := 0x0100 | uint16(c.registers.SP)
	c.bus.WriteByteAt(ptr, (uint8(value >> 8)))
	c.registers.SP--

	ptr = 0x0100 | uint16(c.registers.SP)
	c.bus.WriteByteAt(ptr, (uint8(value & 0xFF)))
	c.registers.SP--
}

// スタック領域からのプル (1バイト)
func (c *CPU) pullByte() uint8 {
	c.registers.SP++
	ptr := 0x0100 | uint16(c.registers.SP)
	return c.bus.ReadByteFrom(ptr)
}

// スタック領域からのプル (2バイト)
func (c *CPU) pullWord() uint16 {
	c.registers.SP++
	ptr := 0x0100 | uint16(c.registers.SP)
	lower := c.bus.ReadByteFrom(ptr)

	c.registers.SP++
	ptr = 0x0100 | uint16(c.registers.SP)
	upper := c.bus.ReadByteFrom(ptr)

	return uint16(upper)<<8 | uint16(lower)
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
	// wramにプログラムを書き込み
	for i := range len(program) {
		c.bus.WriteByteAt(uint16(i), program[i])
	}

	// 実行の無限ループ
	for {
		fmt.Println(c.Trace())
		c.Step()
	}
}

// MARK: CPUのログトレースをとるメソッド
func (c *CPU) Trace() string {
	// 命令の情報を取得
	base := c.registers.PC
	opcode := c.bus.ReadByteFrom(base)
	instruction := c.instructionSet[opcode]

	// オペランドの読み取り
	var operand1, operand2 uint8
	if instruction.Bytes > 1 {
		operand1 = c.bus.ReadByteFrom(base + 1)
	}
	if instruction.Bytes > 2 {
		operand2 = c.bus.ReadByteFrom(base + 2)
	}

	// 16進ダンプの組み立て
	hexDump := fmt.Sprintf("%02X", opcode)
	switch instruction.Bytes {
	case 2:
		hexDump = fmt.Sprintf("%02X %02X", opcode, operand1)
	case 3:
		hexDump = fmt.Sprintf("%02X %02X %02X", opcode, operand1, operand2)
	}
	hexDump = fmt.Sprintf("%-8s", hexDump)

	// オペランド文字列の組み立て
	var operandString string
	var effectiveAddress uint16

	switch instruction.AddressingMode {
	case Implied:
	case Accumulator:
		operandString = "A"
	case Immediate:
		operandString = fmt.Sprintf("#$%02X", operand1)
	case Relative:
		offset := int8(operand1)
		target := base + 2 + uint16(offset)
		operandString = fmt.Sprintf("$%04X", target)
	case ZeroPage:
		effectiveAddress = uint16(operand1)
		operandString = fmt.Sprintf(
			"$%02X = %02X",
			operand1,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case ZeroPageXIndexed:
		base := operand1
		effectiveAddress = uint16(uint8(base + c.registers.X))
		operandString = fmt.Sprintf(
			"$%02X,X @ %02X = %02X",
			base,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case ZeroPageYIndexed:
		base := operand1
		effectiveAddress = uint16(uint8(base + c.registers.Y))
		operandString = fmt.Sprintf(
			"$%02X,Y @ %02X = %02X",
			base,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case Absolute:
		effectiveAddress = uint16(operand1) | (uint16(operand2) << 8)
		if instruction.Mnemonic == "JMP" || instruction.Mnemonic == "JSR" {
			operandString = fmt.Sprintf("$%04X", effectiveAddress)
		} else {
			operandString = fmt.Sprintf(
				"$%04X = %02X",
				effectiveAddress,
				c.bus.ReadByteFrom(effectiveAddress),
			)
		}
	case AbsoluteXIndexed:
		base := uint16(operand1) | (uint16(operand2) << 8)
		effectiveAddress = base + uint16(c.registers.X)
		operandString = fmt.Sprintf(
			"$%04X,X @ %04X = %02X",
			base,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case AbsoluteYIndexed:
		base := uint16(operand1) | (uint16(operand2) << 8)
		effectiveAddress = base + uint16(c.registers.Y)
		operandString = fmt.Sprintf(
			"$%04X,Y @ %04X = %02X",
			base,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case Indirect:
		ptr := uint16(operand1) | (uint16(operand2) << 8)
		var target uint16
		if ptr&0x00FF == 0x00FF {
			low := c.bus.ReadByteFrom(ptr)
			high := c.bus.ReadByteFrom(ptr & 0xFF00)
			target = uint16(high)<<8 | uint16(low)
		} else {
			target = c.bus.ReadWordFrom(ptr)
		}
		operandString = fmt.Sprintf(
			"($%04X) = %04X",
			ptr,
			target,
		)
	case IndexedIndirect:
		base := operand1
		ptr := uint8(base + c.registers.X)
		low := c.bus.ReadByteFrom(uint16(ptr))
		high := c.bus.ReadByteFrom(uint16(ptr+1) & 0x00FF)
		effectiveAddress = uint16(high)<<8 | uint16(low)
		operandString = fmt.Sprintf(
			"($%02X,X) @ %02X = %04X = %02X",
			base,
			ptr,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	case IndirectIndexed:
		base := operand1
		low := c.bus.ReadByteFrom(uint16(base))
		high := c.bus.ReadByteFrom(uint16(base+1) & 0x00FF)
		baseAddr := uint16(high)<<8 | uint16(low)
		effectiveAddress = baseAddr + uint16(c.registers.Y)
		operandString = fmt.Sprintf(
			"($%02X),Y = %04X @ %04X = %02X",
			base,
			baseAddr,
			effectiveAddress,
			c.bus.ReadByteFrom(effectiveAddress),
		)
	}

	// レジスタ情報の組み立て
	registersInfo := fmt.Sprintf(
		"A:%02X X:%02X Y:%02X P:%02X SP:%02X",
		c.registers.A,
		c.registers.X,
		c.registers.Y,
		c.registers.P.ToByte(),
		c.registers.SP,
	)

	// 行全体の組み立て
	return fmt.Sprintf(
		"%04X  %s %4s %-28s %s",
		base,
		hexDump,
		instruction.Mnemonic,
		operandString,
		registersInfo,
	)
}
