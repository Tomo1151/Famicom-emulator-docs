package ppu

import (
	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	PPU_VRAM_SIZE          = 2 * 1024             // 1024kB
	PPU_PALETTE_TABLE_SIZE = 32                   // 32色 (背景/スプライト各16色ずつ)
	PPU_PRIMARY_OAM_SIZE   = 64 * OAM_SPRITE_SIZE // 64スプライト
	PPU_SECONDARY_OAM_SIZE = 8 * OAM_SPRITE_SIZE  // 8スプライト

	PPU_MEMORY_ADDRESS_MASK    = 0x3FFF // 14ビット
	PPU_NAMETABLE_ADDRESS_MASK = 0x2FFF // ミラーリング直前のアドレス

	TILE_SIZE        = 8 // 1タイルのサイズ (幅，高さ)
	MAX_SPRITE_COUNT = 8 // スプライトの最大同時表示数
)

const (
	SCANLINE_START      = 0
	SCANLINE_POSTRENDER = 240
	SCANLINE_VBLANK     = 241
	SCANLINE_PRERENDER  = 261
	SCANLINE_END        = 340
)

const (
	OAM_SPRITE_Y_POS uint = iota
	OAM_SPRITE_TILE_POS
	OAM_SPRITE_ATTR_POS
	OAM_SPRITE_X_POS

	OAM_SPRITE_SIZE = 4 // 1スプライトあたりのバイト数
)

const (
	SPRITE_ZERO_NOT_FOUND = 0xFF
)

// MARK: PPUの定義
type PPU struct {
	vram         [PPU_VRAM_SIZE]uint8          // Video RAM
	oam          [PPU_PRIMARY_OAM_SIZE]uint8   // Object Attribute Memory
	secondaryOAM [PPU_SECONDARY_OAM_SIZE]uint8 // Secondary OAM
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

	// 描画用シフトレジスタ
	backgroundLatch BackgroundLatch
	backgroundShift BackgroundShiftRegister
	spriteLatch     SpriteLatch
	spriteShifts    [8]SpriteShiftRegister

	mapper mappers.Mapper // カートリッジ (CHR ROM) への参照
	canvas *Canvas        // 描画キャンバスの参照

	dot                uint
	scanline           uint
	oamAddress         uint8
	dataBuffer         uint8
	spriteCount        uint
	spriteZeroIndex    uint
	spriteZeroHitDelay bool

	nmi bool

	frame uint64
}

// MARK: PPUのコンストラクタ
func NewPPU(mapper mappers.Mapper, canvas *Canvas) PPU {
	ppu := PPU{
		control:         NewControlRegister(),
		mask:            NewMaskRegister(),
		status:          NewStatusRegister(),
		t:               NewAddressRegister(),
		v:               NewAddressRegister(),
		x:               NewXRegister(),
		w:               NewWRegister(),
		mapper:          mapper,
		canvas:          canvas,
		dot:             0,
		scanline:        0,
		oamAddress:      0x00,
		dataBuffer:      0x00,
		spriteZeroIndex: SPRITE_ZERO_NOT_FOUND,
		frame:           0,
	}

	return ppu
}

// MARK: PPUクロックの更新
func (p *PPU) Tick(cycles uint) {
	for range cycles {
		isRenderingEnabled := p.mask.backgroundEnable || p.mask.spriteEnable

		// スプライト0ヒットは遅延する
		if p.spriteZeroHitDelay {
			p.status.SetSpriteZeroHit(true)
			p.spriteZeroHitDelay = false
		}

		// 可視スキャンラインではセカンダリOAMのクリアと次ラインで使用するスプライトの評価を行う
		if SCANLINE_START <= p.scanline && p.scanline < SCANLINE_POSTRENDER {
			if p.dot == 1 {
				p.clearSecondaryOAM()
			}
			if 65 <= p.dot && p.dot <= 256 {
				p.evaluateNextLineSprite()
			}

			if isRenderingEnabled {

				// シフトレジスタから1ピクセル分を塗る
				if 1 <= p.dot && p.dot <= 256 {
					p.renderPixel()

					// スプライト用シフトレジスタをシフト
					for i := range p.spriteShifts {
						if p.spriteShifts[i].xDistance > 0 {
							// ドットが進む分X距離を更新
							p.spriteShifts[i].xDistance--
						} else {
							// 表示座標であればシフト
							p.spriteShifts[i].shift()
						}
					}
				}

				// 可視領域と321ドット以降では8サイクル毎にVレジスタの水平アドレスをインクリメントする
				if (1 <= p.dot && p.dot <= 256) || (321 <= p.dot && p.dot <= 336) {
					// 背景用シフトレジスタをシフト
					p.backgroundShift.shift()

					// 背景のフェッチ
					p.fetchBackground()

					if p.dot%TILE_SIZE == 0 {
						p.v.incrementHorizontal()
					}
				}

				// 可視領域の最終ドットで垂直アドレスをインクリメントする
				if p.dot == 256 {
					p.v.incrementVertical()
				}

				// 可視領域外最初のドットでVレジスタの水平アドレスをTレジスタにコピーする
				if p.dot == 257 {
					p.t.copyHorizontalBitsTo(&p.v)
				}

				// スプライトのフェッチ
				if 257 <= p.dot && p.dot <= 320 {
					spriteIndex := (p.dot - 257) / 8 // 期間を均等に分割
					p.fetchSprite(spriteIndex)
				}
			}
		}

		// ポストレンダーラインの次のラインではVBlankフラグがセットされる
		if p.scanline == SCANLINE_VBLANK {
			if p.dot == 1 {
				p.status.SetVBlank(true)
				if p.control.GenerateNMI() {
					p.nmi = true
				}
			}
		}

		// プリレンダーラインでは各種フラグがクリアされ，スプライトが評価される
		if p.scanline == SCANLINE_PRERENDER {
			if p.dot == 1 {
				p.status.SetVBlank(false)
				p.status.SetSpriteZeroHit(false)
				p.status.SetSpriteOverflow(false)
				p.spriteZeroHitDelay = false
				p.clearSecondaryOAM()
			}

			// 可視領域と同様にスプライトを評価
			if 65 <= p.dot && p.dot <= 256 {
				p.evaluateNextLineSprite()
			}

			if isRenderingEnabled {
				// 可視領域と321ドット以降では8サイクル毎にVレジスタの水平アドレスをインクリメントする
				if (1 <= p.dot && p.dot <= 256) || (321 <= p.dot && p.dot <= 336) {
					// 背景用シフトレジスタをシフト
					p.backgroundShift.shift()

					p.fetchBackground()

					if p.dot%TILE_SIZE == 0 {
						p.v.incrementHorizontal()
					}
				}

				// 256ドット目で垂直アドレスをインクリメント
				if p.dot == 256 {
					p.v.incrementVertical()
				}

				// 257ドット目で水平アドレスをVレジスタからTレジスタにコピー
				if p.dot == 257 {
					p.t.copyHorizontalBitsTo(&p.v)
				}
			}

			// スプライトのフェッチ
			if 257 <= p.dot && p.dot <= 320 {
				spriteIndex := (p.dot - 257) / 8 // 期間を均等に分割
				p.fetchSprite(spriteIndex)
			}

			// 280 ~ 304ドットでは毎回Vレジスタの垂直アドレスをTレジスタにコピーする
			if isRenderingEnabled && (280 <= p.dot && p.dot <= 304) {
				p.t.copyVerticalBitsTo(&p.v)
			}
		}

		// サイクルとスキャンラインを進める
		p.dot++
		if p.dot > SCANLINE_END {
			p.dot = 0
			p.scanline++

			if p.scanline > SCANLINE_PRERENDER {
				// スキャンラインを0に戻し，フレームを進める
				p.scanline = 0
				p.frame++
			}
		}
	}
}

// MARK: PPUメモリマップの読み取り
func (p *PPU) ReadPPUData() uint8 {
	address := p.v.ToByte() & PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()

	value := p.dataBuffer
	p.dataBuffer = p.ReadPPUVRAM(address)

	if 0x3F00 <= address && address <= 0x3FFF {
		value = p.ReadPPUVRAM(address)
	}

	return value
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
	// fmt.Printf("PPU Scanline: %d, Dot: %d\n", p.scanline, p.dot)
	status := p.status.ToByte()
	p.status.SetVBlank(false) // 読み取りでVBlankフラグとラッチがクリアされる
	p.w.reset()
	return status
}

// MARK: OAMデータの読み取り (CPU: $2004)
func (p *PPU) ReadOAMData() uint8 {
	return p.oam[p.oamAddress]
}

// MARK: PPUメモリマップの読み取り
func (p *PPU) ReadPPUVRAM(address uint16) uint8 {
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

	switch {
	case 0x0000 <= address && address <= 0x1FFF: // パターンテーブル (CHR ROM)
		return p.mapper.ReadCharacterROM(address)
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル (VRAM)
		vramAddress := p.mirroredVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		return p.vram[vramAddress]
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
		}
		return p.paletteTable[paletteTableIndex]
	default:
		return 0x00
	}
}

// MARK: PPUメモリマップへの書き込み
func (p *PPU) WritePPUData(value uint8) {
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
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
		}
		p.paletteTable[paletteTableIndex] = value
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

	if p.w.latch {
		// 2回目の書き込み時はTレジスタをVレジスタにコピー
		p.t.copyAllBitsTo(&p.v)
	}

	p.w.toggle()
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

// MARK: セカンダリOAMのクリア
func (p *PPU) clearSecondaryOAM() {
	for i := range p.secondaryOAM {
		p.secondaryOAM[i] = 0xFF
	}
	p.spriteZeroIndex = SPRITE_ZERO_NOT_FOUND
}

// MARK: 次のスキャンラインで描画するスプライトの評価
func (p *PPU) evaluateNextLineSprite() {
	p.spriteCount = 0
	spriteHeight := p.control.SpriteSize()

	// 次のスキャンラインを計算，プリレンダーライン以降は次のフレームの0ライン目にする
	nextScanline := p.scanline + 1
	if nextScanline > SCANLINE_PRERENDER {
		nextScanline = 0
	}

	for i := range len(p.oam) / OAM_SPRITE_SIZE {
		primaryBase := uint(i) * OAM_SPRITE_SIZE
		spriteY := uint(p.oam[primaryBase+OAM_SPRITE_Y_POS]) + 1 // OAMのY座標は表示座標-1のため補正

		// 次のラインに重なっているか判定
		if nextScanline < spriteY || nextScanline >= spriteY+uint(spriteHeight) {
			continue
		}

		// 重なっているスプライトをプライマリOAMからセカンダリOAMへロード
		if p.spriteCount < MAX_SPRITE_COUNT {
			// プリマリOAMの先頭であればスプライト0として位置を記憶
			if i == 0 {
				p.spriteZeroIndex = p.spriteCount
			}

			secondaryBase := p.spriteCount * OAM_SPRITE_SIZE
			p.secondaryOAM[secondaryBase+OAM_SPRITE_Y_POS] = p.oam[primaryBase+OAM_SPRITE_Y_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_TILE_POS] = p.oam[primaryBase+OAM_SPRITE_TILE_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_ATTR_POS] = p.oam[primaryBase+OAM_SPRITE_ATTR_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_X_POS] = p.oam[primaryBase+OAM_SPRITE_X_POS]

			p.spriteCount++
		} else {
			// 9個目が存在する場合スプライトオーバーフローフラグをセット
			p.status.SetSpriteOverflow(true)
			break
		}
	}
}

// MARK: Vレジスタから描画中の行・列のネームテーブルのアドレスを取得
func (p *PPU) getNameTableAddress() uint16 {
	/*
		基準となるネームテーブルの始点 ($2000 | $2400 | $2800 | $2C00) と
		Vレジスタの下位12ビット(NN YYYYY XXXXX)の論理和で現在の行・列のアドレスが求まる
	*/
	return 0x2000 | (p.v.ToByte() & 0x0FFF)
}

// MARK: Vレジスタから描画中の属性テーブルのアドレスを取得
func (p *PPU) getAttributeTableAddress() uint16 {
	/*
		ベースアドレス: $23C0 (ネームテーブルの末尾)
		p.v & 0x0C00: どの画面（ネームテーブル）かを選択
		(v >> 4) & 0x38: Y座標 (行) をメタタイル単位(4行ごと)に変換
		(v >> 2) & 0x07: X座標 (列) をメタタイル単位(4列ごと)に変換
	*/

	v := p.v.ToByte()
	return 0x23C0 | (v & 0x0C00) | ((v >> 4) & 0x38) | ((v >> 2) & 0x07)
}

// MARK: Vレジスタから描画中の背景タイルのパターンテーブルのアドレスを取得
func (p *PPU) getBackgroundPatternAddress(isUpper bool) uint16 {
	base := p.control.BackgroundPatternTableAddress()
	tile := p.backgroundLatch.nameTable
	fineY := (p.v.ToByte() >> 12) & 0x07
	offset := 0
	if isUpper {
		offset = TILE_SIZE
	}
	return base + (uint16(tile) * TILE_SIZE * 2) + fineY + uint16(offset)
}

// MARK: 描画中のスプライトタイルのパターンテーブルのアドレスを取得
func (p *PPU) getSpritePatternAddress(index uint, isUpper bool) uint16 {
	secondaryBase := index * OAM_SPRITE_SIZE
	spriteY := p.secondaryOAM[secondaryBase+OAM_SPRITE_Y_POS] + 1 // OAMのY座標は表示座標-1のため補正
	spriteTile := p.secondaryOAM[secondaryBase+OAM_SPRITE_TILE_POS]
	attributes := p.secondaryOAM[secondaryBase+OAM_SPRITE_ATTR_POS]
	spriteHeight := p.control.SpriteSize()

	targetScanline := p.scanline
	// 画面外の場合は次スキャンラインを対象にする
	if 257 <= p.dot && p.dot <= 320 {
		targetScanline++
		// プリレンダーラインを超えた場合次フレームの0ライン目にする
		if targetScanline > SCANLINE_PRERENDER {
			targetScanline = 0
		}
	}

	tileY := uint16(targetScanline - uint(spriteY))

	flipV := ((attributes >> 7) & 0x01) == 1
	if flipV {
		tileY = (uint16(spriteHeight) - 1) - tileY
	}

	offset := 0
	if isUpper {
		offset = TILE_SIZE
	}

	var base uint16
	if spriteHeight == TILE_SIZE {
		// 8x8の場合
		base = p.control.SpritePatternTableAddress()
	} else {
		// 8x16の場合
		base = uint16(spriteTile&0x01) * 0x1000 // タイル番号のビット0がネームテーブル選択
		spriteTile = spriteTile & 0xFE
		if tileY >= TILE_SIZE {
			spriteTile++
			tileY -= TILE_SIZE
		}
	}

	return base + (uint16(spriteTile) * TILE_SIZE * 2) + tileY + uint16(offset)
}

// MARK: 背景ネームテーブル/属性テーブルのフェッチ
func (p *PPU) fetchBackground() {
	/*
		フェッチタイミング

		| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
		-----------------------------------
		| - |  NT   |  AT   | BGlow | BGHi  |
	*/

	switch p.dot % TILE_SIZE {
	case 1: // ネームテーブルのフェッチ
		nameTableAddress := p.getNameTableAddress()
		tile := p.ReadPPUVRAM(nameTableAddress)
		p.backgroundLatch.nameTable = tile
	case 3: // 属性テーブルのフェッチ
		attributeTableAddress := p.getAttributeTableAddress()
		attribute := p.ReadPPUVRAM(attributeTableAddress)

		// Vレジスタの位置に応じて該当する2ビットを抽出する
		shift := ((p.v.coarseY>>1)&1)<<2 | ((p.v.coarseX>>1)&1)<<1
		p.backgroundLatch.attribute = (attribute >> shift) & 0x03
	case 5: // パターンテーブル(下位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(false)
		pattern := p.ReadPPUVRAM(patternTableAddress)
		p.backgroundLatch.patternLower = pattern
	case 7: // パターンテーブル(上位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(true)
		pattern := p.ReadPPUVRAM(patternTableAddress)
		p.backgroundLatch.patternUpper = pattern
	case 0: // ラッチからシフトレジスタへロード
		p.backgroundShift.load(&p.backgroundLatch)
	}
}

// MARK: スプライトネームテーブルのフェッチ
func (p *PPU) fetchSprite(index uint) {
	/*
		フェッチタイミング

		| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
		-----------------------------------
		| - |   -   |   -   | Splow | SpHi  |
	*/

	// 使用していない範囲のシフトレジスタをリセット
	if p.spriteCount <= index {
		p.spriteShifts[index].reset()
		return
	}

	switch p.dot % TILE_SIZE {
	case 5: // パターンテーブル(下位)のフェッチ
		address := p.getSpritePatternAddress(index, false)
		p.spriteLatch.patternLower = p.ReadPPUVRAM(address)
	case 7: // パターンテーブル(上位)のフェッチ
		address := p.getSpritePatternAddress(index, true)
		p.spriteLatch.patternUpper = p.ReadPPUVRAM(address)
	case 0: // ラッチからシフトレジスタへロード
		basePtr := index * OAM_SPRITE_SIZE

		spriteX := p.secondaryOAM[basePtr+OAM_SPRITE_X_POS]
		attributes := p.secondaryOAM[basePtr+OAM_SPRITE_ATTR_POS]
		flipH := (attributes>>6)&1 == 1
		p.spriteShifts[index].load(&p.spriteLatch, flipH)
		p.spriteShifts[index].attributes = attributes
		p.spriteShifts[index].xDistance = spriteX
	}
}

// MARK: 背景ピクセルの取得
func (p *PPU) getBackgroundPixel() (pixel uint8, attribute uint8) {
	// 背景描画が無効の場合は 0 を返す
	if !p.mask.backgroundEnable {
		return 0x00, 0x00
	}

	// 左端8ピクセルの背景描画が無効でX座標が8より小さい場合は 0 を返す
	screenX := p.dot - 1
	if screenX < TILE_SIZE && !p.mask.leftmostBackgroundEnable {
		return 0x00, 0x00
	}

	shift := (TILE_SIZE*2 - 1) - p.x.fineX

	// ピクセルの upper / lower の見ているビットを取り出し
	bit0 := (p.backgroundShift.patternLower >> uint16(shift)) & 0x01
	bit1 := (p.backgroundShift.patternUpper >> uint16(shift)) & 0x01
	pixel = uint8((bit1 << 1) | bit0)

	// 属性情報の upper / lower の見ているビットを取り出し
	attrBit0 := (p.backgroundShift.attributeLower >> uint16(shift)) & 0x01
	attrBit1 := (p.backgroundShift.attributeUpper >> uint16(shift)) & 0x01
	attribute = uint8((attrBit1 << 1) | attrBit0)

	return pixel, attribute
}

// MARK: スプライトピクセルの取得
func (p *PPU) getSpritePixel() (pixel uint8, attributes uint8, isSpriteZero bool, priority uint8) {
	// スプライト描画が無効の場合は 0 を返す
	if !p.mask.spriteEnable {
		return 0x00, 0x00, false, 0x00
	}

	// 左端8ピクセルのスプライト描画が無効でX座標が8より小さい場合は 0 を返す
	screenX := p.dot - 1
	if screenX < TILE_SIZE && !p.mask.leftmostSpriteEnable {
		return 0x00, 0x00, false, 0x00
	}

	shift := TILE_SIZE - 1
	for i := range len(p.spriteShifts) {
		if p.spriteShifts[i].xDistance == 0 {
			// シフトレジスタから1ビットを取り出し
			bit0 := (p.spriteShifts[i].patternLower >> shift) & 0x01
			bit1 := (p.spriteShifts[i].patternUpper >> shift) & 0x01
			pixel = uint8((bit1 << 1) | bit0)

			// 透明なピクセルであれば無視
			if pixel == 0 {
				continue
			}

			// 透明でなければこのピクセルの情報を返す
			attributes = p.spriteShifts[i].attributes
			isSpriteZero = (i == int(p.spriteZeroIndex))
			priority = (attributes >> 5) & 0x01
			return pixel, attributes, isSpriteZero, priority
		}
	}
	return 0x00, 0x00, false, 0x00
}

// MARK: ピクセルの書き込み
func (p *PPU) renderPixel() {
	screenX := p.dot - 1
	screenY := p.scanline

	bgPattern, bgAttribute := p.getBackgroundPixel()
	spPattern, spAttributes, isSpriteZero, spPriority := p.getSpritePixel()

	bgOpaque := (bgPattern != 0)
	spOpaque := (spPattern != 0)

	// スプライト0ヒット判定
	if isSpriteZero && bgOpaque && spOpaque && screenX < SCREEN_WIDTH-1 {
		p.spriteZeroHitDelay = true
	}

	// 優先順位に基づいて色を決定
	var color rgb
	switch {
	case !bgOpaque && !spOpaque: // 両方透明
		color = PALETTE[p.paletteTable[0x00]]
	case !bgOpaque && spOpaque: // 背景のみ透明
		color = p.getSpriteColor(spAttributes, spPattern)
	case bgOpaque && !spOpaque: // スプライトのみ透明
		color = p.getBackgroundColor(bgAttribute, bgPattern)
	case bgOpaque && spOpaque: // 両方不透明
		// スプライト優先の場合
		if spPriority == 0 {
			color = p.getSpriteColor(spAttributes, spPattern)
		} else {
			color = p.getBackgroundColor(bgAttribute, bgPattern)
		}
	}

	// 決定した色を描画
	p.canvas.SetPixel(screenX, screenY, color)
}

// MARK: 属性情報とピクセルの値から色を取得するメソッド
func (p *PPU) getBackgroundColor(attribute uint8, pattern uint8) rgb {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		paletteIndex := p.paletteTable[0x00]
		return PALETTE[paletteIndex]
	}

	paletteTableIndex := ((attribute & 0x03) << 2) + pattern
	paletteIndex := p.paletteTable[paletteTableIndex]
	return PALETTE[paletteIndex]
}

// MARK: 属性情報とピクセルの値からスプライトの色を取得するメソッド
func (p *PPU) getSpriteColor(attributes uint8, pattern uint8) rgb {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		paletteIndex := p.paletteTable[0x00]
		return PALETTE[paletteIndex]
	}

	// スプライトパレットの開始位置 0x10 をオフセットとして加算
	paletteTableIndex := 0x10 + ((attributes & 0x03) << 2) + pattern
	paletteIndex := p.paletteTable[paletteTableIndex]
	return PALETTE[paletteIndex]
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

// MARK: 経過フレーム数を返すメソッド
func (p *PPU) Frame() uint64 {
	return p.frame
}
