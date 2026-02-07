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

	// MARK: AND命令
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

	// MARK: LDA命令
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

	return instructionSet
}
