package bus

import "fc-emu/cartridge"

const (
	CPU_WRAM_SIZE = 2 * 1024 // 2kB

	CPU_ADDRESS_START uint16 = 0x0000
	CPU_ADDRESS_END   uint16 = 0xFFFF
)

// MARK: Busの定義
type Bus struct {
	wram      [CPU_WRAM_SIZE]uint8 // CPUのWRAM
	cartridge *cartridge.Cartridge // カートリッジ
}

// MARK: Busのコンストラクタ
func NewBus(cartridge *cartridge.Cartridge) Bus {
	return Bus{
		cartridge: cartridge,
	}
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

		$6000-$7FFF 0x2000 カートリッジ PRG RAM
		$8000-$FFFF 0x8000 カートリッジ PRG ROM / マッパレジスタ
	*/

	switch {
	case CPU_ADDRESS_START <= address && address <= 0x1FFF:
		return b.wram[address&0x07FF] // 2kBでミラーリング
	case 0x6000 <= address && address <= 0x7FFF:
		return b.cartridge.Mapper().ReadProgramRAM(address)
	case 0x8000 <= address && address <= CPU_ADDRESS_END:
		return b.cartridge.Mapper().ReadProgramROM(address)
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

		$6000-$7FFF 0x2000 カートリッジ PRG RAM
		$8000-$FFFF 0x8000 カートリッジ PRG ROM / マッパレジスタ
	*/

	switch {
	case CPU_ADDRESS_START <= address && address <= 0x1FFF:
		b.wram[address&0x07FF] = value // 2kBでミラーリング
	case 0x6000 <= address && address <= 0x7FFF:
		b.cartridge.Mapper().WriteProgramRAM(address, value)
	case 0x8000 <= address && address <= CPU_ADDRESS_END:
		b.cartridge.Mapper().WriteProgramROM(address, value)
	default:
		// TODO: 正しいコンポーネントに値を書き込む
	}
}

// MARK: メモリへの書き込み (2バイト)
func (b *Bus) WriteWordAt(address uint16, value uint16) {
	lower := uint8(value & 0xFF)
	upper := uint8(value >> 8)
	b.WriteByteAt(address, lower)
	b.WriteByteAt(address+1, upper)
}
