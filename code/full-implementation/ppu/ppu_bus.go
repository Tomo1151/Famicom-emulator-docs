package ppu

import (
	"fc-emu/cartridge/mappers"
)

// PPUBus は PPU 用のメモリバスを管理します
type PPUBus struct {
	vram         [PPU_VRAM_SIZE]uint8          // Video RAM
	paletteTable [PPU_PALETTE_TABLE_SIZE]uint8 // Palette RAM
	mapper       mappers.Mapper                // カートリッジ (CHR ROM) への参照
}

// NewPPUBus は PPUBus のインスタンスを作成します
func NewPPUBus(mapper mappers.Mapper) *PPUBus {
	return &PPUBus{
		mapper: mapper,
	}
}

// ReadByte は PPU アドレス空間から 1 バイト読み取ります
func (pb *PPUBus) ReadByte(address uint16) uint8 {
	/*
		PPU メモリマップ
		(範囲 / サイズ / 対象)

		$0000-$1FFF 0x2000 パターンテーブル
		$2000-$2FFF 0x1000 ネームテーブル
		$3000-$3EFF 0x0F00 ネームテーブルのミラーリング
		$3F00-$3F1F 0x0020 パレットテーブル
		$3F20-$3FFF 0x00E0 パレットテーブルのミラーリング
		$4000-$FFFF 0x4000 $0000-$3FFF のミラーリング
	*/

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

// WriteByte は PPU アドレス空間へ 1 バイト書き込みます
func (pb *PPUBus) WriteByte(address uint16, value uint8) {
	address &= PPU_MEMORY_ADDRESS_MASK

	switch {
	case PPU_ADDRESS_START <= address && address <= 0x1FFF: // パターンテーブル (CHR RAM)
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

// ReadPalette は パレットテーブルから直接値を取得します (描画用)
func (pb *PPUBus) ReadPalette(index uint8) uint8 {
	paletteTableIndex := index % PPU_PALETTE_TABLE_SIZE
	if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
		paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
	}
	return pb.paletteTable[paletteTableIndex]
}

// mirrorVRAMAddress は VRAMのアドレスをミラーリング設定に基づいて計算します
func (pb *PPUBus) mirrorVRAMAddress(address uint16) uint16 {
	/*
		VRAM ネームテーブル
		(範囲 / サイズ / 対象)

		$2000-$2400 0x0400 画面1
		$2400-$2800 0x0400 画面2
	*/

	vramAddress := address - 0x2000 // 先頭オフセットを引きVRAMのアドレスに変換
	mirroring := pb.mapper.Mirroring()

	/*
		ネームテーブルの位置を求める
		[ 0 ][ 1 ]
		[ 2 ][ 3 ]
	*/
	nameTableIndex := vramAddress / 0x0400

	switch mirroring {
	case mappers.MIRRORING_VERTICAL:
		/*
			[ A ][ B ] $2000 $2400
			[ a ][ b ] $2800 $2C00

			A: $2000
			a: $2800 → $2000
			B: $2400
			b: $2C00 → $2400
		*/

		switch nameTableIndex {
		case 2, 3:
			vramAddress -= 0x0800
		}
	case mappers.MIRRORING_HORIZONTAL:
		/*
			[ A ][ a ] $2000 $2400
			[ B ][ b ] $2800 $2C00

			A: $2000
			a: $2400 → $2000
			B: $2800 → $2400
			b: $2C00 → $2400
		*/

		switch nameTableIndex {
		case 1, 2:
			vramAddress -= 0x0400
		case 3:
			vramAddress -= 0x0800
		}
	}

	return vramAddress
}
