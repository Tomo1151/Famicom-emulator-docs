package bus

import (
	"fc-emu/cartridge"
	"fc-emu/joypad"
	"fc-emu/ppu"
)

const (
	CPU_WRAM_SIZE = 2 * 1024 // 2kB
)

// MARK: Busの定義
type Bus struct {
	wram      [CPU_WRAM_SIZE]uint8 // CPUのWRAM
	cartridge *cartridge.Cartridge // カートリッジ
	ppu       *ppu.PPU             // PPU
	joypad1   *joypad.JoyPad       // コントローラ (1P)
	joypad2   *joypad.JoyPad       // コントローラ (2P)
}

// MARK: Busのコンストラクタ
func NewBus(cartridge *cartridge.Cartridge, ppu *ppu.PPU, joypad1 *joypad.JoyPad, joypad2 *joypad.JoyPad) Bus {
	return Bus{
		cartridge: cartridge,
		ppu:       ppu,
		joypad1:   joypad1,
		joypad2:   joypad2,
	}
}

// MARK: 各コンポーネントのクロックの更新
func (b *Bus) Tick(cycles uint) {
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

		$2000              PPU コントロールレジスタ
		$2001              PPU マスクレジスタ
		$2002              PPU ステータスレジスタ
		$2004              OAM データ
		$2007              PPU データ
		$2008-$3FFF 0x1FF8 PPU I/O レジスタ ($2000-$2008) のミラーリング

		$6000-$7FFF 0x2000 カートリッジ PRG RAM
		$8000-$FFFF 0x8000 カートリッジ PRG ROM / マッパレジスタ
	*/

	switch {
	case 0x0000 <= address && address <= 0x1FFF: // CPU WRAM
		return b.wram[address&0x07FF] // 2kBでミラーリング
	case address == 0x2000: // PPU CTRL
		return b.ppu.ReadPPUControl()
	case address == 0x2001: // PPU MASK
		return b.ppu.ReadPPUMask()
	case address == 0x2002: // PPU STATUS
		return b.ppu.ReadPPUStatus()
	case address == 0x2004: // OAM DATA
		return b.ppu.ReadOAMData()
	case address == 0x2007: // PPU DATA
		return b.ppu.ReadPPUData()
	case 0x2008 <= address && address <= 0x3FFF: // PPU I/O ミラーリング
		ptr := 0x2000 | (address & 0x07) // $2000-$2008 を繰り返す
		return b.ReadByteFrom(ptr)
	case address == 0x4016: // コントローラ (1P)
		return b.joypad1.ReadJoyPad()
	case address == 0x4017: // コントローラ (2P)
		return b.joypad2.ReadJoyPad()
	case 0x6000 <= address && address <= 0x7FFF: // PRG RAM
		return b.cartridge.Mapper().ReadProgramRAM(address)
	case 0x8000 <= address && address <= 0xFFFF: // PRG ROM
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
	case 0x0000 <= address && address <= 0x1FFF: // CPU WRAM
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
	case address == 0x4016: // コントローラ (1P/2P)
		b.joypad1.WriteJoyPad(value)
		b.joypad2.WriteJoyPad(value)
	case 0x6000 <= address && address <= 0x7FFF: // PRG RAM
		b.cartridge.Mapper().WriteProgramRAM(address, value)
	case 0x8000 <= address && address <= 0xFFFF: // PRG ROM
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

// MARK: NMI状態の取得
func (b *Bus) NMI() bool {
	return b.ppu.PollNMI()
}
