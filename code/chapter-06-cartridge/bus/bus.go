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
	/*
		CPU メモリマップ
		(範囲 / サイズ / コンポーネント)

		$0000-$07FF 0x0800 2kBのWRAM
		$0800-$0FFF 0x0800 WRAMのミラーリング x3
		$1000-$17FF 0x0800
		$1800-$1FFF 0x0800
	*/

	switch {
	case 0x0000 <= address && address <= 0x1FFF:
		return b.wram[address&0x07FF] // 2kBでミラーリング
	default:
		// TODO: 正しいコンポーネントから値を読み取って返す
		return 0x00
	}
}

// MARK: メモリの読み取り (2バイト)
func (b *Bus) ReadWordFrom(address uint16) uint16 {
	lower := b.ReadByteFrom(address)
	upper := b.ReadByteFrom(address + 1)
	return uint16(upper)<<8 | uint16(lower)
}

// MARK: メモリへの書き込み (1バイト)
func (b *Bus) WriteByteAt(address uint16, value uint8) {
	/*
		CPU メモリマップ
		(範囲 / サイズ / コンポーネント)

		$0000-$07FF 0x0800 2kBのWRAM
		$0800-$0FFF 0x0800 WRAMのミラーリング x3
		$1000-$17FF 0x0800
		$1800-$1FFF 0x0800
	*/

	switch {
	case 0x0000 <= address && address <= 0x1FFF:
		b.wram[address] = value
	default:
		// TODO: 正しいコンポーネントに値を書き込む
	}
}

// MARK: メモリへの書き込み (2バイト)
func (b *Bus) WriteWordAt(address uint16, value uint8) {
	lower := uint8(value & 0xFF)
	upper := uint8(value >> 8)
	b.WriteByteAt(address, lower)
	b.WriteByteAt(address+1, upper)
}
