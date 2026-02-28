package ppu

import (
	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	PPU_VRAM_SIZE          = 2 * 1024 // 1024kB
	PPU_PALETTE_TABLE_SIZE = 32
	PPU_OAM_SIZE           = 4 * 64

	PPU_MEMORY_ADDRESS_MASK    = 0x3FFF // 14ビット
	PPU_NAMETABLE_ADDRESS_MASK = 0x2FFF

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

	nmi bool
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

// MARK: PPUメモリマップの読み取り
func (p *PPU) ReadPPUMemory() uint8 {
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

	address := p.v.ToByte() & PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()

	switch {
	case 0x0000 <= address && address <= 0x1FFF: // パターンテーブル (CHR ROM)
		value := p.dataBuffer
		p.dataBuffer = p.mapper.ReadCharacterROM(address)
		return value
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル (VRAM)
		value := p.dataBuffer
		vramAddress := p.mirroredVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		p.dataBuffer = p.vram[vramAddress]
		return value
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル
		paletteTableIndex := address - 0x3F00
		return p.paletteTable[paletteTableIndex%PPU_PALETTE_TABLE_SIZE]
	default:
		return 0x00
	}
}

// MARK: PPUコントロールレジスタの読み取り (CPU: $2000)
func (p *PPU) ReadPPUControl() uint8 {
	return p.control.ToByte()
}

// MARK: PPUマスクレジスタの読み取り (CPU: $2001)
func (p *PPU) ReadPPUMask() uint8 {
	return p.mask.ToByte()
}

// MARK: PPUステータスレジスタの読み取り (CPU: $2002)
func (p *PPU) ReadPPUStatus() uint8 {
	status := p.status.ToByte()
	p.status.SetVBlankStatus(false) // 読み取りでVBlankフラグとラッチがクリアされる
	p.w.reset()
	return status
}

// MARK: OAMデータの読み取り (CPU: $2004)
func (p *PPU) ReadOAMData() uint8 {
	return p.oam[p.oamAddress]
}

// MARK: PPUメモリマップへの書き込み
func (p *PPU) WritePPUMemory(value uint8) {
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

	address := p.v.ToByte() & PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()

	switch {
	case 0x0000 <= address && address <= 0x1FFF: // パターンテーブル (CHR RAM)
		if p.mapper.IsCharacterRAM() {
			p.mapper.WriteCharacterRAM(address, value)
		}
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル
		vramAddress := p.mirroredVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		p.vram[vramAddress] = value
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル
		paletteTableIndex := address - 0x3F00
		p.paletteTable[paletteTableIndex%PPU_PALETTE_TABLE_SIZE] = value
	default:
	}
}

// MARK: PPUコントロールレジスタの書き込み (CPU: $2000)
func (p *PPU) WritePPUControl(value uint8) {
	prev := p.control.GenerateNMI()

	p.control.SetFromByte(value)
	p.t.updateNameTable(value)

	// VBlank中にGenerateNMIがセットされたタイミングでNMIが発生
	if !prev && p.control.GenerateNMI() && p.status.VBlank() {
		p.nmi = true
	}
}

// MARK: PPUマスクレジスタの書き込み (CPU: $2001)
func (p *PPU) WritePPUMask(value uint8) {
	p.mask.SetFromByte(value)
}

// MARK: OAMアドレスの書き込み (CPU: $2003)
func (p *PPU) WriteOAMAddress(value uint8) {
	p.oamAddress = value
}

// MARK: OAMデータの書き込み (CPU: $2004)
func (p *PPU) WriteOAMData(value uint8) {
	p.oam[p.oamAddress] = value
	p.oamAddress++ // OAMアドレスは自動でインクリメントされる
}

// MARK: PPUスクロールの書き込み (CPU: $2005)
func (p *PPU) WritePPUScroll(value uint8) {
	if !p.w.latch {
		p.x.update(value) // 1回目はXレジスタも更新 (fineX)
	}
	p.t.updateScroll(value, p.w.latch) // Tレジスタは毎回更新 (fineY / coarseX / coarseY)
	p.w.toggle()
}

// MARK: PPUアドレスの書き込み (CPU: $2006)
func (p *PPU) WritePPUAddress(value uint8) {
	p.t.updateAddress(value, p.w.latch)
	p.w.toggle()

	if p.w.latch {
		// 2回目の書き込み時はTレジスタをVレジスタにコピー
		p.t.copyAllBitsTo(&p.v)
	}
}

// MARK: DMA転送の実行 (CPU: $4014)
func (p *PPU) DMATransfer(bytes *[256]uint8) {
	for _, value := range *bytes {
		p.oam[p.oamAddress] = value
		p.oamAddress++
	}
}

// MARK: ミラーリング後のVRAMアドレスを取得
func (p *PPU) mirroredVRAMAddress(address uint16) uint16 {
	/*
		VRAM ネームテーブル
		(範囲 / サイズ / 対象)

		$2000-$2400 0x0400 画面1
		$2400-$2800 0x0400 画面2
	*/

	vramAddress := address - 0x2000 // 先頭オフセットを引きVRAMのアドレスに変換
	mirroring := p.mapper.Mirroring()

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

// MARK: VRAMアドレスのインクリメント
func (p *PPU) incrementVRAMAddress() {
	step := uint16(p.control.VRAMAddressIncrement())
	address := (p.v.ToByte() + step)
	p.v.SetFromWord(address & PPU_MEMORY_ADDRESS_MASK)
}

// MARK: NMIの状態を取得するメソッド
func (p *PPU) PollNMI() bool {
	if p.nmi {
		p.nmi = false
		return true
	} else {
		return false
	}
}

// MARK: NMIを取得するメソッド
func (p *PPU) NMI() bool {
	return p.nmi
}
