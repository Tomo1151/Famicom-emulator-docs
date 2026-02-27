package ppu

import (
	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	PPU_VRAM_SIZE          = 2 * 1024 // 1024kB
	PPU_PALETTE_TABLE_SIZE = 32
	PPU_OAM_SIZE           = 4 * 64

	TILE_SIZE = 8
)

const (
	SCANLINE_START      = 0
	SCANLINE_POSTRENDER = 240
	SCANLINE_VBLANK     = 241
	SCANLINE_PRERENDER  = 261
	SCANLINE_END        = 341
)

const (
	OAM_SPRITE_Y_POS uint = iota
	OAM_SPRITE_TILE_POS
	OAM_SPRITE_ATTR_POS
	OAM_SPRITE_X_POS
)

// MARK: PPUの定義
type PPU struct {
	vram         [PPU_VRAM_SIZE]uint8          // Video RAM
	oam          [PPU_OAM_SIZE]uint8           // Object Attribute Memory
	paletteTable [PPU_PALETTE_TABLE_SIZE]uint8 // Palette Table

	// IOレジスタ
	control ControlRegister // $2000
	mask    MaskRegister    // $2001
	status  StatusRegister  // $2002

	// 内部レジスタ
	t AddressRegiseter
	v AddressRegiseter
	x XRegister
	w WRegister

	mapper mappers.Mapper // カートリッジ (CHR ROM) への参照

	cycles     uint
	scanline   uint16
	oamAddress uint8
	dataBuffer uint8
}

// MARK: PPUのコンストラクタ
func NewPPU(mapper mappers.Mapper) PPU {
	ppu := PPU{
		control:    NewControlRegister(),
		mask:       NewMaskRegister(),
		status:     NewStatusRegister(),
		t:          NewAddressRegister(),
		v:          NewAddressRegister(),
		x:          NewXRegister(),
		w:          NewWRegister(),
		mapper:     mapper,
		cycles:     0,
		scanline:   0,
		oamAddress: 0x00,
		dataBuffer: 0x00,
	}

	return ppu
}
