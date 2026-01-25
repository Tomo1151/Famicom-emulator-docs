package bus

const (
	CPU_WRAM_SIZE = 2 * 1024 // 2kB
)

// MARK: Busの定義
type Bus struct {
	wram [CPU_WRAM_SIZE]uint8 // 一時的なプログラムのバイト列
}

// MARK: Busのコンストラクタ
func NewBus() Bus {
	return Bus{}
}

// MARK: メモリの読み取り (1バイト)
func (b *Bus) ReadByteFrom(address uint16) uint8 {
	// TODO: 正しいコンポーネントから値を読み取って返す
	return b.wram[address]
}

// MARK: メモリの読み取り (2バイト)
func (b *Bus) ReadWordFrom(address uint16) uint16 {
	lower := b.ReadByteFrom(address)
	upper := b.ReadByteFrom(address + 1)
	return uint16(upper)<<8 | uint16(lower)
}

// MARK: メモリへの書き込み (1バイト)
func (b *Bus) WriteByteAt(address uint16, value uint8) {
	// TODO: 正しいコンポーネントに値を書き込む
	b.wram[address] = value
}

// MARK: メモリへの書き込み (2バイト)
func (b *Bus) WriteWordAt(address uint16, value uint8) {
	lower := uint8(value & 0xFF)
	upper := uint8(value >> 8)
	b.WriteByteAt(address, lower)
	b.WriteByteAt(address+1, upper)
}
