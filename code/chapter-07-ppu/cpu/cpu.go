package cpu

import (
	"fmt"

	"fc-emu/bus"
)

// MARK: CPUの定義
type CPU struct {
	registers      registers
	bus            bus.Bus
	instructionSet instructionSet
	cycles         uint
}

// MARK: CPUのコンストラクタ
func NewCPU(bus bus.Bus) *CPU {
	cpu := &CPU{}

	cpu.bus = bus
	cpu.instructionSet = generateInstructionSet(cpu)
	cpu.registers = registers{
		A:  0x00,
		X:  0x00,
		Y:  0x00,
		SP: 0xFD,
		PC: cpu.bus.ReadWordFrom(0xFFFC),
		P:  NewStatusRegister(),
	}
	cpu.cycles = 0

	// FIXME: nestest用のエントリポイント
	// cpu.registers.PC = 0xC000
	return cpu
}

// MARK: 命令サイクルの実行メソッド
func (c *CPU) Step() {
	// 命令のフェッチ
	opcode := c.bus.ReadByteFrom(c.registers.PC)
	c.registers.PC++

	// 命令のデコード
	instruction := c.instructionSet[opcode]

	// 命令の実行
	instruction.Handler(instruction.AddressingMode)

	// PCを書き換えない命令のみ，命令長の分プログラムカウンタを進める（オペコード分 - 1）
	isJump := instruction.Mnemonic == "JMP" ||
		instruction.Mnemonic == "JSR" || instruction.Mnemonic == "RTI" ||
		instruction.Mnemonic == "RTS" || instruction.Mnemonic == "BRK"

	if !isJump {
		c.registers.PC += uint16(instruction.Bytes - 1)
	}

	// 各コンポーネントのクロック
	c.cycles += uint(instruction.Cycles)
	c.bus.Tick(uint(instruction.Cycles))

	if c.bus.NMI() {
		c.interrupt(NMI)
	}
}

// MARK: NMIのハンドリング
func (c *CPU) interrupt(interrupt Interrupt) {
	// 現在のPCを退避
	c.pushWord(c.registers.PC)

	// ステータスレジスタをスタックにプッシュ
	status := c.registers.P
	status.Break = interrupt.BFlagMask&0b0001_0000 != 0
	status.Reserved = interrupt.BFlagMask&0b0010_0000 != 0
	c.pushByte(status.ToByte())
	c.registers.P.IrqDisabled = true

	c.bus.Tick(uint(interrupt.CPUCycles))
	c.registers.PC = c.bus.ReadWordFrom(interrupt.VectorAddress)
}

// MARK: 実行メソッド
func (c *CPU) Run() {
	for {
		fmt.Println(c.Trace())
		c.Step()
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

// MARK: 算術演算系 公式命令
// ADC命令の実装
func (c *CPU) adc(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	var carry uint16 = 0
	if c.registers.P.Carry {
		carry = 1
	}
	sum := uint16(c.registers.A) + uint16(value) + carry
	result := uint8(sum)

	c.registers.P.Carry = sum > 0xFF
	c.registers.P.Overflow = ((c.registers.A ^ result) & (value ^ result) & 0x80) != 0
	c.registers.A = result
	c.updateNZFlags(c.registers.A)
}

// DEC命令の実装
func (c *CPU) dec(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address) - 1
	c.bus.WriteByteAt(address, value)
	c.updateNZFlags(value)
}

// DEX命令の実装
func (c *CPU) dex(_ AddressingMode) {
	c.registers.X--
	c.updateNZFlags(c.registers.X)
}

// DEY命令の実装
func (c *CPU) dey(_ AddressingMode) {
	c.registers.Y--
	c.updateNZFlags(c.registers.Y)
}

// INC命令の実装
func (c *CPU) inc(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address) + 1
	c.bus.WriteByteAt(address, value)
	c.updateNZFlags(value)
}

// INX命令の実装
func (c *CPU) inx(_ AddressingMode) {
	c.registers.X++
	c.updateNZFlags(c.registers.X)
}

// INY命令の実装
func (c *CPU) iny(_ AddressingMode) {
	c.registers.Y++
	c.updateNZFlags(c.registers.Y)
}

// SBC命令の実装
func (c *CPU) sbc(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	inverted := ^value

	var carry uint16 = 0
	if c.registers.P.Carry {
		carry = 1
	}

	sum := uint16(c.registers.A) + uint16(inverted) + carry
	result := uint8(sum)

	c.registers.P.Carry = sum > 0xFF
	c.registers.P.Overflow = ((c.registers.A ^ result) & (inverted ^ result) & 0x80) != 0
	c.registers.A = result
	c.updateNZFlags(c.registers.A)
}

// MARK: ビット演算系 公式命令
// AND命令の実装
func (c *CPU) and(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A &= value
	c.updateNZFlags(c.registers.A)
}

// BIT命令の実装
func (c *CPU) bit(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)

	c.registers.P.Zero = (value & c.registers.A) == 0
	c.registers.P.Overflow = (value & (1 << STATUS_REG_OVERFLOW_POS)) != 0
	c.registers.P.Negative = (value & (1 << STATUS_REG_NEGATIVE_POS)) != 0
}

// EOR命令の実装
func (c *CPU) eor(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A ^= value
	c.updateNZFlags(c.registers.A)
}

// ORA命令の実装
func (c *CPU) ora(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A |= value
	c.updateNZFlags(c.registers.A)
}

// MARK: ビットシフト系 公式命令
// ASL命令の実装
func (c *CPU) asl(mode AddressingMode) {
	if mode == Accumulator {
		c.registers.P.Carry = (c.registers.A >> 7) != 0
		c.registers.A = c.registers.A << 1
		c.updateNZFlags(c.registers.A)
	} else {
		address := c.calcOperandAddress(mode)
		value := c.bus.ReadByteFrom(address)
		c.registers.P.Carry = (value >> 7) != 0
		value <<= 1
		c.bus.WriteByteAt(address, value)
		c.updateNZFlags(value)
	}
}

// LSR命令の実装
func (c *CPU) lsr(mode AddressingMode) {
	if mode == Accumulator {
		c.registers.P.Carry = (c.registers.A & 0x01) != 0
		c.registers.A >>= 1
		c.updateNZFlags(c.registers.A)
	} else {
		address := c.calcOperandAddress(mode)
		value := c.bus.ReadByteFrom(address)
		c.registers.P.Carry = (value & 0x01) != 0
		value >>= 1
		c.bus.WriteByteAt(address, value)
		c.updateNZFlags(value)
	}
}

// ROL命令の実装
func (c *CPU) rol(mode AddressingMode) {
	if mode == Accumulator {
		carry := (c.registers.A >> 7) != 0
		c.registers.A <<= 1
		if c.registers.P.Carry {
			c.registers.A |= 0x01
		}
		c.registers.P.Carry = carry
		c.updateNZFlags(c.registers.A)
	} else {
		address := c.calcOperandAddress(mode)
		value := c.bus.ReadByteFrom(address)
		carry := (value >> 7) != 0
		value <<= 1
		if c.registers.P.Carry {
			value |= 0x01
		}
		c.bus.WriteByteAt(address, value)

		c.registers.P.Carry = carry
		c.updateNZFlags(value)
	}
}

// ROR命令の実装
func (c *CPU) ror(mode AddressingMode) {
	if mode == Accumulator {
		carry := (c.registers.A & 0x01) != 0
		c.registers.A >>= 1
		if c.registers.P.Carry {
			c.registers.A |= (1 << 7)
		}
		c.registers.P.Carry = carry
		c.updateNZFlags(c.registers.A)
	} else {
		address := c.calcOperandAddress(mode)
		value := c.bus.ReadByteFrom(address)
		carry := (value & 0x01) != 0
		value >>= 1
		if c.registers.P.Carry {
			value |= (1 << 7)
		}
		c.bus.WriteByteAt(address, value)
		c.registers.P.Carry = carry
		c.updateNZFlags(value)
	}
}

// MARK: 条件分岐系 公式命令
// BCC命令の実装
func (c *CPU) bcc(mode AddressingMode) {
	if !c.registers.P.Carry {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BCC命令の実装
func (c *CPU) bcs(mode AddressingMode) {
	if c.registers.P.Carry {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BEQ命令の実装
func (c *CPU) beq(mode AddressingMode) {
	if c.registers.P.Zero {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BMI命令の実装
func (c *CPU) bmi(mode AddressingMode) {
	if c.registers.P.Negative {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BNE命令の実装
func (c *CPU) bne(mode AddressingMode) {
	if !c.registers.P.Zero {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BPL命令の実装
func (c *CPU) bpl(mode AddressingMode) {
	if !c.registers.P.Negative {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BVC命令の実装
func (c *CPU) bvc(mode AddressingMode) {
	if !c.registers.P.Overflow {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// BVS命令の実装
func (c *CPU) bvs(mode AddressingMode) {
	if c.registers.P.Overflow {
		address := c.calcOperandAddress(mode)
		c.registers.PC = address
	}
}

// MARK: ジャンプ系 公式命令
// BRK命令の実装
func (c *CPU) brk(_ AddressingMode) {
	c.pushWord(c.registers.PC + 1)

	status := c.registers.P
	status.Break = true
	c.pushByte(status.ToByte())

	c.registers.P.IrqDisabled = true
	c.registers.PC = c.bus.ReadWordFrom(0xFFFE)
}

// JMP命令の実装
func (c *CPU) jmp(mode AddressingMode) {
	c.registers.PC = c.calcOperandAddress(mode)
}

// JSR命令の実装
func (c *CPU) jsr(mode AddressingMode) {
	c.pushWord(c.registers.PC + 1) // オペランド部の後半アドレスをプッシュ
	c.registers.PC = c.calcOperandAddress(mode)
}

// RTI命令の実装
func (c *CPU) rti(_ AddressingMode) {
	status := c.pullByte()
	mask := uint8((1 << STATUS_REG_BREAK_POS) | (1 << STATUS_REG_RESERVED_POS))
	c.registers.P.SetFromByte((status & ^mask) | (c.registers.P.ToByte() & mask))
	c.registers.PC = c.pullWord()
}

// RTS命令の実装
func (c *CPU) rts(_ AddressingMode) {
	c.registers.PC = c.pullWord() + 1
}

// MARK: フラグ操作系 公式命令
// CLC命令の実装
func (c *CPU) clc(_ AddressingMode) {
	c.registers.P.Carry = false
}

// CLD命令の実装
func (c *CPU) cld(_ AddressingMode) {
	c.registers.P.Decimal = false
}

// CLI命令の実装
func (c *CPU) cli(_ AddressingMode) {
	c.registers.P.IrqDisabled = false
}

// CLV命令の実装
func (c *CPU) clv(_ AddressingMode) {
	c.registers.P.Overflow = false
}

// SEC命令の実装
func (c *CPU) sec(_ AddressingMode) {
	c.registers.P.Carry = true
}

// SED命令の実装
func (c *CPU) sed(_ AddressingMode) {
	c.registers.P.Decimal = true
}

// SEI命令の実装
func (c *CPU) sei(_ AddressingMode) {
	c.registers.P.IrqDisabled = true
}

// MARK: 比較系 公式命令
// CMP命令の実装
func (c *CPU) cmp(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.P.Carry = c.registers.A >= value
	c.updateNZFlags(c.registers.A - value)
}

// CPX命令の実装
func (c *CPU) cpx(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.P.Carry = c.registers.X >= value
	c.updateNZFlags(c.registers.X - value)
}

// CPY命令の実装
func (c *CPU) cpy(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.P.Carry = c.registers.Y >= value
	c.updateNZFlags(c.registers.Y - value)
}

// MARK: データアクセス系 公式命令
// LDA命令の実装
func (c *CPU) lda(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A = value
	c.updateNZFlags(c.registers.A)
}

// LDX命令の実装
func (c *CPU) ldx(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.X = value
	c.updateNZFlags(c.registers.X)
}

// LDY命令の実装
func (c *CPU) ldy(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.Y = value
	c.updateNZFlags(c.registers.Y)
}

// STA命令の実装
func (c *CPU) sta(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	c.bus.WriteByteAt(address, c.registers.A)
}

// STX命令の実装
func (c *CPU) stx(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	c.bus.WriteByteAt(address, c.registers.X)
}

// STY命令の実装
func (c *CPU) sty(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	c.bus.WriteByteAt(address, c.registers.Y)
}

// MARK: スタック操作系 公式命令
// PHA命令の実装
func (c *CPU) pha(_ AddressingMode) {
	c.pushByte(c.registers.A)
}

// PHP命令の実装
func (c *CPU) php(_ AddressingMode) {
	c.pushByte(c.registers.P.ToByte() | (1 << STATUS_REG_BREAK_POS))
}

// PLA命令の実装
func (c *CPU) pla(_ AddressingMode) {
	c.registers.A = c.pullByte()
	c.updateNZFlags(c.registers.A)
}

// PLP命令の実装
func (c *CPU) plp(_ AddressingMode) {
	value := c.pullByte()
	mask := uint8((1 << STATUS_REG_BREAK_POS) | (1 << STATUS_REG_RESERVED_POS))
	c.registers.P.SetFromByte((value & ^mask) | (c.registers.P.ToByte() & mask))
}

// MARK: データ転送系 公式命令
// TAX命令の実装
func (c *CPU) tax(_ AddressingMode) {
	c.registers.X = c.registers.A
	c.updateNZFlags(c.registers.X)
}

// TAY命令の実装
func (c *CPU) tay(_ AddressingMode) {
	c.registers.Y = c.registers.A
	c.updateNZFlags(c.registers.Y)
}

// TSX命令の実装
func (c *CPU) tsx(_ AddressingMode) {
	c.registers.X = c.registers.SP
	c.updateNZFlags(c.registers.X)
}

// TXA命令の実装
func (c *CPU) txa(_ AddressingMode) {
	c.registers.A = c.registers.X
	c.updateNZFlags(c.registers.A)
}

// TXS命令の実装
func (c *CPU) txs(_ AddressingMode) {
	c.registers.SP = c.registers.X
}

// TYA命令の実装
func (c *CPU) tya(_ AddressingMode) {
	c.registers.A = c.registers.Y
	c.updateNZFlags(c.registers.A)
}

// NOP命令の実装
func (c *CPU) nop(_ AddressingMode) {
}

// MARK: 非公式命令
// ALR命令の実装 (ASR)
func (c *CPU) alr(mode AddressingMode) {
	c.and(mode)
	c.lsr(Accumulator)
}

// ANC命令の実装 (AAC)
func (c *CPU) anc(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A &= value
	c.updateNZFlags(c.registers.A)
	c.registers.P.Carry = c.registers.P.Negative
}

// ARR命令の実装 (ARR)
func (c *CPU) arr(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A &= value
	c.registers.A >>= 1
	if c.registers.P.Carry {
		c.registers.A |= (1 << 7)
	}

	c.registers.P.Carry = (c.registers.A >> 6) != 0
	c.registers.P.Overflow = ((c.registers.A >> 6) & 1) != ((c.registers.A >> 5) & 1) // XOR
	c.updateNZFlags(c.registers.A)
}

// AXS命令の実装 (SBX / SAX)
func (c *CPU) axs(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.X &= c.registers.A

	c.registers.P.Carry = c.registers.X >= value
	c.registers.X -= value
	c.updateNZFlags(c.registers.X)
}

// LXA命令の実装 (ATX / OAL)
func (c *CPU) lxa(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A &= value
	c.registers.X = c.registers.A
	c.updateNZFlags(c.registers.X)
}

// SAX命令の実装 (AAX / AXS)
func (c *CPU) sax(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	result := c.registers.X & c.registers.A
	c.bus.WriteByteAt(address, result)
}

// AHX命令の実装 (AXA / SHA)
func (c *CPU) ahx(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	result := (c.registers.X & c.registers.A) & 7
	c.bus.WriteByteAt(address, result)
}

// DCP命令の実装 (DCP)
func (c *CPU) dcp(mode AddressingMode) {
	c.dec(mode)
	c.cmp(mode)
}

// ISC命令の実装 (ISB / INS)
func (c *CPU) isc(mode AddressingMode) {
	c.inc(mode)
	c.sbc(mode)
}

// LAS命令の実装 (LAR / LAE)
func (c *CPU) las(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	result := c.registers.SP & value

	c.registers.A = result
	c.registers.X = result
	c.registers.SP = result
	c.updateNZFlags(result)
}

// LAX命令の実装 (LAX)
func (c *CPU) lax(mode AddressingMode) {
	c.lda(mode)
	c.tax(mode)
}

// RLA命令の実装 (RLA)
func (c *CPU) rla(mode AddressingMode) {
	c.rol(mode)
	c.and(mode)
}

// RRA命令の実装 (RRA)
func (c *CPU) rra(mode AddressingMode) {
	c.ror(mode)
	c.adc(mode)
}

// SLO命令の実装 (ASO)
func (c *CPU) slo(mode AddressingMode) {
	c.asl(mode)
	c.ora(mode)
}

// SRE命令の実装 (LSE)
func (c *CPU) sre(mode AddressingMode) {
	c.lsr(mode)
	c.eor(mode)
}

// TAS命令の実装 (SHS)
func (c *CPU) tas(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	c.registers.SP = (c.registers.X & c.registers.A)
	result := c.registers.SP & (uint8(address>>8) + 1)
	c.bus.WriteByteAt(address, result)
}

// SHX命令の実装 (SXA / XAS)
func (c *CPU) shx(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	result := c.registers.X & (uint8(address>>8) + 1)
	c.bus.WriteByteAt(address, result)
}

// SHY命令の実装 (SYA / SAY)
func (c *CPU) shy(mode AddressingMode) {
	address := c.calcOperandAddress(mode)
	result := c.registers.Y & (uint8(address>>8) + 1)
	c.bus.WriteByteAt(address, result)
}

// KIL命令の実装 (JAM / HLT)
func (c *CPU) kil(_ AddressingMode) {
}

// DOP命令の実装 (NOP / SKB / SKW)
func (c *CPU) dop(_ AddressingMode) {
}

// TOP命令の実装 (NOP / IGN)
func (c *CPU) top(_ AddressingMode) {
}

// XAA命令の実装 (ANE)
func (c *CPU) xaa(mode AddressingMode) {
	// NOTE: 未定義動作
	address := c.calcOperandAddress(mode)
	value := c.bus.ReadByteFrom(address)
	c.registers.A = (c.registers.A | 0xEE) & c.registers.X & value
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
			if 0x2000 <= effectiveAddress && effectiveAddress <= 0x3FFF {
				operandString = fmt.Sprintf(
					"$%04X = PPU IO",
					effectiveAddress,
				)
			} else {
				operandString = fmt.Sprintf(
					"$%04X = %02X",
					effectiveAddress,
					c.bus.ReadByteFrom(effectiveAddress),
				)
			}
		}
	case AbsoluteXIndexed:
		base := uint16(operand1) | (uint16(operand2) << 8)
		effectiveAddress = base + uint16(c.registers.X)
		if 0x2000 <= effectiveAddress && effectiveAddress <= 0x3FFF {
			operandString = fmt.Sprintf(
				"$%04X,X @ %04X = PPU IO",
				base,
				effectiveAddress,
			)
		} else {
			operandString = fmt.Sprintf(
				"$%04X,X @ %04X = %02X",
				base,
				effectiveAddress,
				c.bus.ReadByteFrom(effectiveAddress),
			)
		}
	case AbsoluteYIndexed:
		base := uint16(operand1) | (uint16(operand2) << 8)
		effectiveAddress = base + uint16(c.registers.Y)
		if 0x2000 <= effectiveAddress && effectiveAddress <= 0x3FFF {
			operandString = fmt.Sprintf(
				"$%04X,Y @ %04X = PPU IO",
				base,
				effectiveAddress,
			)
		} else {
			operandString = fmt.Sprintf(
				"$%04X,Y @ %04X = %02X",
				base,
				effectiveAddress,
				c.bus.ReadByteFrom(effectiveAddress),
			)
		}
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
		"%04X  %s %4s %-27s %s",
		base,
		hexDump,
		instruction.Mnemonic,
		operandString,
		registersInfo,
	)
}
