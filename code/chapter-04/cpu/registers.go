package cpu

const (
	STATUS_REG_NEGATIVE_POS uint8 = iota
	STATUS_REG_OVERFLOW_POS
	STATUS_REG_RESERVED_POS
	STATUS_REG_BREAK_POS
	STATUS_REG_DECIMAL_POS
	STATUS_REG_IRQDISABLED_POS
	STATUS_REG_ZERO_POS
	STATUS_REG_CARRY_POS
)

// MARK: CPUレジスタの定義
type registers struct {
	A  uint8
	X  uint8
	Y  uint8
	SP uint8
	PC uint16
	P  statusRegister
}

// MARK: CPU ステータスレジスタの定義
type statusRegister struct {
	Negative    bool
	Overflow    bool
	Reserved    bool // 常にtrue
	Break       bool
	Decimal     bool
	IrqDisabled bool
	Zero        bool
	Carry       bool
}

// MARK: CPUステータスレジスタのコンストラクタ
func NewStatusRegister() statusRegister {
	return statusRegister{
		Negative:    false,
		Overflow:    false,
		Reserved:    true,
		Break:       false,
		Decimal:     true,
		IrqDisabled: false,
		Zero:        false,
		Carry:       false,
	}
}

// MARK: ステータスレジスタ構造体からuint8へ変換するメソッド
func (sr *statusRegister) ToByte() uint8 {
	var value uint8 = 0x00

	if sr.Negative {
		value |= 1 << STATUS_REG_NEGATIVE_POS
	}
	if sr.Overflow {
		value |= 1 << STATUS_REG_OVERFLOW_POS
	}
	if sr.Reserved {
		value |= 1 << STATUS_REG_RESERVED_POS
	}
	if sr.Break {
		value |= 1 << STATUS_REG_BREAK_POS
	}
	if sr.Decimal {
		value |= 1 << STATUS_REG_DECIMAL_POS
	}
	if sr.IrqDisabled {
		value |= 1 << STATUS_REG_IRQDISABLED_POS
	}
	if sr.Zero {
		value |= 1 << STATUS_REG_ZERO_POS
	}
	if sr.Carry {
		value |= 1 << STATUS_REG_CARRY_POS
	}

	return value
}

// MARK: uint8からステータスレジスタ構造体へ値を反映するメソッド
func (sr *statusRegister) SetFromByte(value uint8) {
	sr.Negative = (value & (1 << STATUS_REG_NEGATIVE_POS)) != 0
	sr.Overflow = (value & (1 << STATUS_REG_OVERFLOW_POS)) != 0
	sr.Reserved = (value & (1 << STATUS_REG_RESERVED_POS)) != 0
	sr.Break = (value & (1 << STATUS_REG_BREAK_POS)) != 0
	sr.Decimal = (value & (1 << STATUS_REG_DECIMAL_POS)) != 0
	sr.IrqDisabled = (value & (1 << STATUS_REG_IRQDISABLED_POS)) != 0
	sr.Zero = (value & (1 << STATUS_REG_ZERO_POS)) != 0
	sr.Carry = (value & (1 << STATUS_REG_CARRY_POS)) != 0
}
