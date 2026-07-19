package bus

import (
	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	PPU_VRAM_SIZE          = 2 * 1024 // 2kBのVRAM
	PPU_PALETTE_TABLE_SIZE = 32       // 32色

	PPU_ADDRESS_START = 0x0000
	PPU_ADDRESS_END   = 0xFFFF

	PPU_MEMORY_ADDRESS_MASK    = 0x3FFF // 14ビット
	PPU_NAMETABLE_ADDRESS_MASK = 0x2FFF // ミラーリング直前のアドレス
)

// MARK: PPUBusの定義
type PPUBus struct {
	vram         [PPU_VRAM_SIZE]uint8          // Video RAM
	paletteTable [PPU_PALETTE_TABLE_SIZE]uint8 // Palette RAM
	mapper       mappers.Mapper                // カートリッジ (CHR ROM) への参照
}

// MARK: PPUBusのコンストラクタ
func NewPPUBus(mapper mappers.Mapper) *PPUBus {
	return &PPUBus{
		mapper: mapper,
	}
}

// MARK: メモリの読み取り (1バイト)
func (pb *PPUBus) ReadByteFrom(address uint16) uint8 {
	// PPUアドレスは14ビットでマスクされる
	address &= PPU_MEMORY_ADDRESS_MASK

	switch {
	case PPU_ADDRESS_START <= address && address <= 0x1FFF: // パターンテーブル (CHR ROM)
		return pb.mapper.ReadCharacterROM(address)
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル (VRAM)
		vramAddress := pb.mirrorVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		return pb.vram[vramAddress]
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル (Palette RAM)
		return pb.ReadPalette(uint8(address - 0x3F00))
	default:
		return 0x00
	}
}

// MARK: メモリへの書き込み (1バイト)
func (pb *PPUBus) WriteByteAt(address uint16, value uint8) {
	// PPUアドレスは14ビットでマスクされる
	address &= PPU_MEMORY_ADDRESS_MASK

	switch {
	case PPU_ADDRESS_START <= address && address <= 0x1FFF: // パターンテーブル (CHR RAM)
		// キャラクタRAMが有効であれば書き込み
		if pb.mapper.IsCharacterRAM() {
			pb.mapper.WriteCharacterRAM(address, value)
		}
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル (VRAM)
		vramAddress := pb.mirrorVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		pb.vram[vramAddress] = value
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル (Palette RAM)
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
		}
		pb.paletteTable[paletteTableIndex] = value
	}
}

// MARK: パレットの読み取りメソッド
func (pb *PPUBus) ReadPalette(index uint8) uint8 {
	paletteTableIndex := index % PPU_PALETTE_TABLE_SIZE
	if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
		paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
	}
	return pb.paletteTable[paletteTableIndex]
}

// ミラーリングを考慮したVRAMアドレスの計算メソッド
func (pb *PPUBus) mirrorVRAMAddress(address uint16) uint16 {
	vramAddress := address - 0x2000 // 先頭オフセットを引きVRAM配列のインデックスに変換
	mirroring := pb.mapper.Mirroring()

	nameTableIndex := vramAddress / 0x0400

	switch mirroring {
	case mappers.MIRRORING_VERTICAL:
		switch nameTableIndex {
		case 2, 3:
			vramAddress -= 0x0800
		}
	case mappers.MIRRORING_HORIZONTAL:
		switch nameTableIndex {
		case 1, 2:
			vramAddress -= 0x0400
		case 3:
			vramAddress -= 0x0800
		}
	}

	return vramAddress
}

// MARK: Mapperの取得メソッド
func (pb *PPUBus) Mapper() mappers.Mapper {
	return pb.mapper
}
