package cpu

// MARK: アドレッシングモードの定義
type AddressingMode uint8

const (
	Implied          AddressingMode = iota // impl
	Accumulator                            // A
	Immediate                              // #
	ZeroPage                               //zpg
	ZeroPageXIndexed                       // zpg,X
	ZeroPageYIndexed                       // zpg,Y
	Absolute                               // abs
	AbsoluteXIndexed                       // abs,X
	AbsoluteYIndexed                       // abs,Y
	Relative                               // rel
	Indirect                               // ind
	IndexedIndirect                        // X,ind
	IndirectIndexed                        // ind,Y
)

// MARK: 命令の定義
type instruction struct {
	Mnemonic       string
	Opcode         uint8
	AddressingMode AddressingMode
	Bytes          uint8
	Cycles         uint8
	Handler        func(mode AddressingMode)
}

// MARK: 命令セットの定義
type instructionSet map[uint8]instruction

// MARK: 命令セットの生成関数
func generateInstructionSet(c *CPU) instructionSet {
	instructionSet := make(instructionSet)

	// MARK: 算術演算系 公式命令
	// ADC命令

	instructionSet[0x69] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x69,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.adc,
	}

	instructionSet[0x65] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x65,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.adc,
	}

	instructionSet[0x75] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x75,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.adc,
	}

	instructionSet[0x6D] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x6D,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.adc,
	}

	instructionSet[0x7D] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x7D,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.adc,
	}

	instructionSet[0x79] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x79,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.adc,
	}

	instructionSet[0x61] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x61,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.adc,
	}

	instructionSet[0x71] = instruction{
		Mnemonic:       "ADC",
		Opcode:         0x71,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.adc,
	}

	// DEC命令
	instructionSet[0xC6] = instruction{
		Mnemonic:       "DEC",
		Opcode:         0xC6,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.dec,
	}

	instructionSet[0xD6] = instruction{
		Mnemonic:       "DEC",
		Opcode:         0xD6,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.dec,
	}

	instructionSet[0xCE] = instruction{
		Mnemonic:       "DEC",
		Opcode:         0xCE,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.dec,
	}

	instructionSet[0xDE] = instruction{
		Mnemonic:       "DEC",
		Opcode:         0xDE,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.dec,
	}

	// DEX命令
	instructionSet[0xCA] = instruction{
		Mnemonic:       "DEX",
		Opcode:         0xCA,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.dex,
	}

	// DEY命令
	instructionSet[0x88] = instruction{
		Mnemonic:       "DEY",
		Opcode:         0x88,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.dey,
	}

	// INC命令
	instructionSet[0xE6] = instruction{
		Mnemonic:       "INC",
		Opcode:         0xE6,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.inc,
	}

	instructionSet[0xF6] = instruction{
		Mnemonic:       "INC",
		Opcode:         0xF6,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.inc,
	}

	instructionSet[0xEE] = instruction{
		Mnemonic:       "INC",
		Opcode:         0xEE,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.inc,
	}

	instructionSet[0xFE] = instruction{
		Mnemonic:       "INC",
		Opcode:         0xFE,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.inc,
	}

	// INX命令
	instructionSet[0xE8] = instruction{
		Mnemonic:       "INX",
		Opcode:         0xE8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.inx,
	}

	// INY命令
	instructionSet[0xC8] = instruction{
		Mnemonic:       "INY",
		Opcode:         0xC8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.iny,
	}

	// SBC命令
	instructionSet[0xE9] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xE9,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.sbc,
	}

	instructionSet[0xE5] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xE5,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.sbc,
	}

	instructionSet[0xF5] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xF5,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.sbc,
	}

	instructionSet[0xED] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xED,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.sbc,
	}

	instructionSet[0xFD] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xFD,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.sbc,
	}

	instructionSet[0xF9] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xF9,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.sbc,
	}

	instructionSet[0xE1] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xE1,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.sbc,
	}

	instructionSet[0xF1] = instruction{
		Mnemonic:       "SBC",
		Opcode:         0xF1,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.sbc,
	}

	// MARK: ビット演算系 公式命令
	// AND命令
	instructionSet[0x29] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x29,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.and,
	}

	instructionSet[0x25] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x25,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.and,
	}

	instructionSet[0x35] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x35,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.and,
	}

	instructionSet[0x2D] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x2D,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.and,
	}

	instructionSet[0x3D] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x3D,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.and,
	}

	instructionSet[0x39] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x39,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.and,
	}

	instructionSet[0x21] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x21,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.and,
	}

	instructionSet[0x31] = instruction{
		Mnemonic:       "AND",
		Opcode:         0x31,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.and,
	}

	// BIT命令
	instructionSet[0x24] = instruction{
		Mnemonic:       "BIT",
		Opcode:         0x24,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.bit,
	}

	instructionSet[0x2C] = instruction{
		Mnemonic:       "BIT",
		Opcode:         0x2C,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.bit,
	}

	// EOR命令
	instructionSet[0x49] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x49,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.eor,
	}

	instructionSet[0x45] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x45,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.eor,
	}

	instructionSet[0x55] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x55,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.eor,
	}

	instructionSet[0x4D] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x4D,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.eor,
	}

	instructionSet[0x5D] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x5D,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.eor,
	}

	instructionSet[0x59] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x59,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.eor,
	}

	instructionSet[0x41] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x41,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.eor,
	}

	instructionSet[0x51] = instruction{
		Mnemonic:       "EOR",
		Opcode:         0x51,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.eor,
	}

	// ORA命令
	instructionSet[0x09] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x09,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.ora,
	}

	instructionSet[0x05] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x05,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.ora,
	}

	instructionSet[0x15] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x15,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.ora,
	}

	instructionSet[0x0D] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x0D,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ora,
	}

	instructionSet[0x1D] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x1D,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ora,
	}

	instructionSet[0x19] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x19,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ora,
	}

	instructionSet[0x01] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x01,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.ora,
	}

	instructionSet[0x11] = instruction{
		Mnemonic:       "ORA",
		Opcode:         0x11,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.ora,
	}

	// MARK: ビットシフト系 公式命令
	// ASL命令
	instructionSet[0x0A] = instruction{
		Mnemonic:       "ASL",
		Opcode:         0x0A,
		AddressingMode: Accumulator,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.asl,
	}

	instructionSet[0x06] = instruction{
		Mnemonic:       "ASL",
		Opcode:         0x06,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.asl,
	}

	instructionSet[0x16] = instruction{
		Mnemonic:       "ASL",
		Opcode:         0x16,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.asl,
	}

	instructionSet[0x0E] = instruction{
		Mnemonic:       "ASL",
		Opcode:         0x0E,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.asl,
	}

	instructionSet[0x1E] = instruction{
		Mnemonic:       "ASL",
		Opcode:         0x1E,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.asl,
	}

	// LSR命令
	instructionSet[0x4A] = instruction{
		Mnemonic:       "LSR",
		Opcode:         0x4A,
		AddressingMode: Accumulator,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.lsr,
	}

	instructionSet[0x46] = instruction{
		Mnemonic:       "LSR",
		Opcode:         0x46,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.lsr,
	}

	instructionSet[0x56] = instruction{
		Mnemonic:       "LSR",
		Opcode:         0x56,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.lsr,
	}

	instructionSet[0x4E] = instruction{
		Mnemonic:       "LSR",
		Opcode:         0x4E,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.lsr,
	}

	instructionSet[0x5E] = instruction{
		Mnemonic:       "LSR",
		Opcode:         0x5E,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.lsr,
	}

	// ROL命令
	instructionSet[0x2A] = instruction{
		Mnemonic:       "ROL",
		Opcode:         0x2A,
		AddressingMode: Accumulator,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.rol,
	}

	instructionSet[0x26] = instruction{
		Mnemonic:       "ROL",
		Opcode:         0x26,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.rol,
	}

	instructionSet[0x36] = instruction{
		Mnemonic:       "ROL",
		Opcode:         0x36,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.rol,
	}

	instructionSet[0x2E] = instruction{
		Mnemonic:       "ROL",
		Opcode:         0x2E,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.rol,
	}

	instructionSet[0x3E] = instruction{
		Mnemonic:       "ROL",
		Opcode:         0x3E,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.rol,
	}

	// ROR命令
	instructionSet[0x6A] = instruction{
		Mnemonic:       "ROR",
		Opcode:         0x6A,
		AddressingMode: Accumulator,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.ror,
	}

	instructionSet[0x66] = instruction{
		Mnemonic:       "ROR",
		Opcode:         0x66,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.ror,
	}

	instructionSet[0x76] = instruction{
		Mnemonic:       "ROR",
		Opcode:         0x76,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.ror,
	}

	instructionSet[0x6E] = instruction{
		Mnemonic:       "ROR",
		Opcode:         0x6E,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.ror,
	}

	instructionSet[0x7E] = instruction{
		Mnemonic:       "ROR",
		Opcode:         0x7E,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         7,
		Handler:        c.ror,
	}

	// MARK: 条件分岐系 公式命令
	// BCC命令
	instructionSet[0x90] = instruction{
		Mnemonic:       "BCC",
		Opcode:         0x90,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bcc,
	}

	// BCS命令
	instructionSet[0xB0] = instruction{
		Mnemonic:       "BCS",
		Opcode:         0xB0,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bcs,
	}

	// BEQ命令
	instructionSet[0xF0] = instruction{
		Mnemonic:       "BEQ",
		Opcode:         0xF0,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.beq,
	}

	// BMI命令
	instructionSet[0x30] = instruction{
		Mnemonic:       "BMI",
		Opcode:         0x30,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bmi,
	}

	// BNE命令
	instructionSet[0xD0] = instruction{
		Mnemonic:       "BNE",
		Opcode:         0xD0,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bne,
	}

	// BPL命令
	instructionSet[0x10] = instruction{
		Mnemonic:       "BPL",
		Opcode:         0x10,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bpl,
	}

	// BVC命令
	instructionSet[0x50] = instruction{
		Mnemonic:       "BCS",
		Opcode:         0x50,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bvc,
	}

	// BVS命令
	instructionSet[0x70] = instruction{
		Mnemonic:       "BVS",
		Opcode:         0x70,
		AddressingMode: Relative,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.bvs,
	}

	// MARK: ジャンプ系 公式命令
	// BRK命令
	instructionSet[0x00] = instruction{
		Mnemonic:       "BRK",
		Opcode:         0x00,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         7,
		Handler:        c.brk,
	}

	// JMP命令
	instructionSet[0x4C] = instruction{
		Mnemonic:       "JMP",
		Opcode:         0x4C,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         3,
		Handler:        c.jmp,
	}

	instructionSet[0x6C] = instruction{
		Mnemonic:       "JMP",
		Opcode:         0x6C,
		AddressingMode: Indirect,
		Bytes:          3,
		Cycles:         5,
		Handler:        c.jmp,
	}

	// JSR命令
	instructionSet[0x20] = instruction{
		Mnemonic:       "JSR",
		Opcode:         0x20,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         6,
		Handler:        c.jsr,
	}

	// RTI命令
	instructionSet[0x40] = instruction{
		Mnemonic:       "RTI",
		Opcode:         0x40,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         6,
		Handler:        c.rti,
	}

	// RTS命令
	instructionSet[0x60] = instruction{
		Mnemonic:       "RTS",
		Opcode:         0x60,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         6,
		Handler:        c.rts,
	}

	// MARK: フラグ操作系 公式命令
	// CLC命令
	instructionSet[0x18] = instruction{
		Mnemonic:       "CLC",
		Opcode:         0x18,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.clc,
	}

	// CLD命令
	instructionSet[0xD8] = instruction{
		Mnemonic:       "CLD",
		Opcode:         0xD8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.cld,
	}

	// CLI命令
	instructionSet[0x58] = instruction{
		Mnemonic:       "CLI",
		Opcode:         0x58,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.cli,
	}

	// CLV命令
	instructionSet[0xB8] = instruction{
		Mnemonic:       "CLV",
		Opcode:         0xB8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.clv,
	}

	// SEC命令
	instructionSet[0x38] = instruction{
		Mnemonic:       "SEC",
		Opcode:         0x38,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.sec,
	}

	// SED命令
	instructionSet[0xF8] = instruction{
		Mnemonic:       "SED",
		Opcode:         0xF8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.sed,
	}

	// SEI命令
	instructionSet[0x78] = instruction{
		Mnemonic:       "SEI",
		Opcode:         0x78,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.sei,
	}

	// MARK: 比較系 公式命令
	// CMP命令
	instructionSet[0xC9] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xC9,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.cmp,
	}

	instructionSet[0xC5] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xC5,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.cmp,
	}

	instructionSet[0xD5] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xD5,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.cmp,
	}

	instructionSet[0xCD] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xCD,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.cmp,
	}

	instructionSet[0xDD] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xDD,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.cmp,
	}

	instructionSet[0xD9] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xD9,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.cmp,
	}

	instructionSet[0xC1] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xC1,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.cmp,
	}

	instructionSet[0xD1] = instruction{
		Mnemonic:       "CMP",
		Opcode:         0xD1,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.cmp,
	}

	// CPX命令
	instructionSet[0xE0] = instruction{
		Mnemonic:       "CPX",
		Opcode:         0xE0,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.cpx,
	}

	instructionSet[0xE4] = instruction{
		Mnemonic:       "CPX",
		Opcode:         0xE4,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.cpx,
	}

	instructionSet[0xEC] = instruction{
		Mnemonic:       "CPX",
		Opcode:         0xEC,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.cpx,
	}

	// CPY命令
	instructionSet[0xC0] = instruction{
		Mnemonic:       "CPY",
		Opcode:         0xC0,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.cpy,
	}

	instructionSet[0xC4] = instruction{
		Mnemonic:       "CPY",
		Opcode:         0xC4,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.cpy,
	}

	instructionSet[0xCC] = instruction{
		Mnemonic:       "CPY",
		Opcode:         0xCC,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.cpy,
	}

	// MARK: データアクセス系 公式命令
	// LDA命令
	instructionSet[0xA9] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xA9,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.lda,
	}

	instructionSet[0xA5] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xA5,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.lda,
	}

	instructionSet[0xB5] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xB5,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.lda,
	}

	instructionSet[0xAD] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xAD,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.lda,
	}

	instructionSet[0xBD] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xBD,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.lda,
	}

	instructionSet[0xB9] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xB9,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.lda,
	}

	instructionSet[0xA1] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xA1,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.lda,
	}

	instructionSet[0xB1] = instruction{
		Mnemonic:       "LDA",
		Opcode:         0xB1,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         5,
		Handler:        c.lda,
	}

	// LDX命令
	instructionSet[0xA2] = instruction{
		Mnemonic:       "LDX",
		Opcode:         0xA2,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.ldx,
	}

	instructionSet[0xA6] = instruction{
		Mnemonic:       "LDX",
		Opcode:         0xA6,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.ldx,
	}

	instructionSet[0xB6] = instruction{
		Mnemonic:       "LDX",
		Opcode:         0xB6,
		AddressingMode: ZeroPageYIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.ldx,
	}

	instructionSet[0xAE] = instruction{
		Mnemonic:       "LDX",
		Opcode:         0xAE,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ldx,
	}

	instructionSet[0xBE] = instruction{
		Mnemonic:       "LDX",
		Opcode:         0xBE,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ldx,
	}

	// LDY命令
	instructionSet[0xA0] = instruction{
		Mnemonic:       "LDY",
		Opcode:         0xA0,
		AddressingMode: Immediate,
		Bytes:          2,
		Cycles:         2,
		Handler:        c.ldy,
	}

	instructionSet[0xA4] = instruction{
		Mnemonic:       "LDY",
		Opcode:         0xA4,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.ldy,
	}

	instructionSet[0xB4] = instruction{
		Mnemonic:       "LDY",
		Opcode:         0xB4,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.ldy,
	}

	instructionSet[0xAC] = instruction{
		Mnemonic:       "LDY",
		Opcode:         0xAC,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ldy,
	}

	instructionSet[0xBC] = instruction{
		Mnemonic:       "LDY",
		Opcode:         0xBC,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.ldy,
	}

	// STA命令
	instructionSet[0x85] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x85,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.sta,
	}

	instructionSet[0x95] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x95,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.sta,
	}

	instructionSet[0x8D] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x8D,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.sta,
	}

	instructionSet[0x9D] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x9D,
		AddressingMode: AbsoluteXIndexed,
		Bytes:          3,
		Cycles:         5,
		Handler:        c.sta,
	}

	instructionSet[0x99] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x99,
		AddressingMode: AbsoluteYIndexed,
		Bytes:          3,
		Cycles:         5,
		Handler:        c.sta,
	}

	instructionSet[0x81] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x81,
		AddressingMode: IndexedIndirect,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.sta,
	}

	instructionSet[0x91] = instruction{
		Mnemonic:       "STA",
		Opcode:         0x91,
		AddressingMode: IndirectIndexed,
		Bytes:          2,
		Cycles:         6,
		Handler:        c.sta,
	}

	// STX命令
	instructionSet[0x86] = instruction{
		Mnemonic:       "STX",
		Opcode:         0x86,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.stx,
	}

	instructionSet[0x96] = instruction{
		Mnemonic:       "STX",
		Opcode:         0x96,
		AddressingMode: ZeroPageYIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.stx,
	}

	instructionSet[0x8E] = instruction{
		Mnemonic:       "STX",
		Opcode:         0x8E,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.stx,
	}

	// STY命令
	instructionSet[0x84] = instruction{
		Mnemonic:       "STY",
		Opcode:         0x84,
		AddressingMode: ZeroPage,
		Bytes:          2,
		Cycles:         3,
		Handler:        c.sty,
	}

	instructionSet[0x94] = instruction{
		Mnemonic:       "STY",
		Opcode:         0x94,
		AddressingMode: ZeroPageXIndexed,
		Bytes:          2,
		Cycles:         4,
		Handler:        c.sty,
	}

	instructionSet[0x8C] = instruction{
		Mnemonic:       "STY",
		Opcode:         0x8C,
		AddressingMode: Absolute,
		Bytes:          3,
		Cycles:         4,
		Handler:        c.sty,
	}

	// MARK: スタック操作系 公式命令
	// PHA命令
	instructionSet[0x48] = instruction{
		Mnemonic:       "PHA",
		Opcode:         0x48,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         3,
		Handler:        c.pha,
	}

	// PHP命令
	instructionSet[0x08] = instruction{
		Mnemonic:       "PHP",
		Opcode:         0x08,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         3,
		Handler:        c.php,
	}

	// PLA命令
	instructionSet[0x68] = instruction{
		Mnemonic:       "PLA",
		Opcode:         0x68,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         4,
		Handler:        c.pla,
	}

	// PLP命令
	instructionSet[0x28] = instruction{
		Mnemonic:       "PLP",
		Opcode:         0x28,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         4,
		Handler:        c.plp,
	}

	// MARK: データ転送系 公式命令
	// TAX命令
	instructionSet[0xAA] = instruction{
		Mnemonic:       "TAX",
		Opcode:         0xAA,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.tax,
	}

	// TAY命令
	instructionSet[0xA8] = instruction{
		Mnemonic:       "TAY",
		Opcode:         0xA8,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.tay,
	}

	// TSX命令
	instructionSet[0xBA] = instruction{
		Mnemonic:       "TSX",
		Opcode:         0xBA,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.tsx,
	}

	// TXA命令
	instructionSet[0x8A] = instruction{
		Mnemonic:       "TXA",
		Opcode:         0x8A,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.txa,
	}

	// TXS命令
	instructionSet[0x9A] = instruction{
		Mnemonic:       "TXS",
		Opcode:         0x9A,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.txs,
	}

	// TYA命令
	instructionSet[0x98] = instruction{
		Mnemonic:       "TYA",
		Opcode:         0x98,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.tya,
	}

	// NOP命令
	instructionSet[0xEA] = instruction{
		Mnemonic:       "NOP",
		Opcode:         0xEA,
		AddressingMode: Implied,
		Bytes:          1,
		Cycles:         2,
		Handler:        c.nop,
	}

	return instructionSet
}
