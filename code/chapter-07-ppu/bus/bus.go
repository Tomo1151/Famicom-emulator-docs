package bus

import (
	"fc-emu/cartridge/mappers"
	"fc-emu/ppu"
)

const (
	CPU_WRAM_SIZE = 2 * 1024 // 2kB

	CPU_ADDRESS_START uint16 = 0x0000
	CPU_ADDRESS_END   uint16 = 0xFFFF
)

// MARK: Busの定義
type Bus struct {
	wram   [CPU_WRAM_SIZE]uint8 // CPUのWRAM
	mapper mappers.Mapper       // カートリッジの参照
	ppu    *ppu.PPU             // PPU

	cycles uint // CPUサイクル
}

// MARK: Busのコンストラクタ
func NewBus(mapper mappers.Mapper, ppu *ppu.PPU) Bus {
	return Bus{
		mapper: mapper,
		ppu:    ppu,
		cycles: 0,
	}
}

// MARK: 各コンポーネントのクロックの更新
func (b *Bus) Tick(cycles uint) {
	b.cycles += cycles
	b.ppu.Tick(cycles * 3)
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

		$2002              PPU ステータスレジスタ
		$2004              OAM データ
		$2007              PPU データ
		$2008-$3FFF 0x1FF8 PPU I/O レジスタ ($2000-$2008) のミラーリング

		$6000-$7FFF 0x2000 カートリッジ PRG RAM
		$8000-$FFFF 0x8000 カートリッジ PRG ROM / マッパレジスタ
	*/

	switch {
	case CPU_ADDRESS_START <= address && address <= 0x1FFF: // CPU WRAM
		return b.wram[address&0x07FF] // 2kBでミラーリング
	case address == 0x2002: // PPU STATUS
		return b.ppu.ReadPPUStatus()
	case address == 0x2004: // OAM DATA
		return b.ppu.ReadOAMData()
	case address == 0x2007: // PPU DATA
		return b.ppu.ReadPPUData()
	case 0x2008 <= address && address <= 0x3FFF: // PPU I/O ミラーリング
		ptr := 0x2000 | (address & 0x07) // $2000-$2008 を繰り返す
		return b.ReadByteFrom(ptr)
	case 0x6000 <= address && address <= 0x7FFF: // PRG RAM
		return b.mapper.ReadProgramRAM(address)
	case 0x8000 <= address && address <= CPU_ADDRESS_END: // PRG ROM
		return b.mapper.ReadProgramROM(address)
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

		$2000              PPU コントロールレジスタ
		$2001              PPU マスクレジスタ
		$2003              OAM アドレス
		$2004              OAM アドレス
		$2005              OAM データ
		$2006              PPU アドレス
		$2007              PPU データ
		$2008-$3FFF 0x1FF8 PPU I/O レジスタ ($2000-$2008) のミラーリング
		$4014              OAM DMA転送

		$6000-$7FFF 0x2000 カートリッジ PRG RAM
		$8000-$FFFF 0x8000 カートリッジ PRG ROM / マッパレジスタ
	*/

	switch {
	case CPU_ADDRESS_START <= address && address <= 0x1FFF: // CPU WRAM
		b.wram[address&0x07FF] = value // 2kBでミラーリング
	case address == 0x2000: // PPU CTRL
		b.ppu.WritePPUControl(value)
	case address == 0x2001: // PPU MASK
		b.ppu.WritePPUMask(value)
	case address == 0x2003: // OAM ADDR
		b.ppu.WriteOAMAddress(value)
	case address == 0x2004: // OAM DATA
		b.ppu.WriteOAMData(value)
	case address == 0x2005: // PPU SCROLL
		b.ppu.WritePPUScroll(value)
	case address == 0x2006: // PPU ADDR
		b.ppu.WritePPUAddress(value)
	case address == 0x2007: // PPU DATA
		b.ppu.WritePPUData(value)
	case 0x2008 <= address && address <= 0x3FFF: // PPU I/O ミラーリング
		ptr := 0x2000 | (address & 0x07) // $2000-$2008 を繰り返す
		b.WriteByteAt(ptr, value)
	case address == 0x4014: // OAM DMA転送
		var buffer [256]uint8
		upper := uint16(value) << 8
		for i := range 256 {
			buffer[i] = b.ReadByteFrom(upper | uint16(i))
		}
		b.ppu.DMATransfer(&buffer)

		// DMA転送は513/514サイクルを消費する
		dmaCycles := uint(513)
		if b.cycles%2 != 0 {
			dmaCycles = 514
		}
		b.Tick(dmaCycles)
	case 0x6000 <= address && address <= 0x7FFF: // PRG RAM
		b.mapper.WriteProgramRAM(address, value)
	case 0x8000 <= address && address <= CPU_ADDRESS_END: // PRG ROM
		b.mapper.WriteProgramROM(address, value)
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

// MARK: NMI状態の取得
func (b *Bus) NMI() bool {
	return b.ppu.PollNMI()
}
