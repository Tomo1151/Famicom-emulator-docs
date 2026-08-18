package ppu

import (
	"github.com/veandco/go-sdl2/sdl"

	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	PPU_VRAM_SIZE          = 2 * 1024             // 2048 kB: 2画面
	PPU_PALETTE_TABLE_SIZE = 32                   // 32 B: 32色 (背景/スプライト各16色ずつ)
	PPU_PRIMARY_OAM_SIZE   = 64 * OAM_SPRITE_SIZE // 256 B: 64スプライト
	PPU_SECONDARY_OAM_SIZE = 8 * OAM_SPRITE_SIZE  // 32 B:8スプライト

	PPU_ADDRESS_START      = 0x0000
	PPU_ADDRESS_END        = 0xFFFF
	PPU_VRAM_ADDRESS_SPACE = 0x4000

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
	COLOR_EMPHASIZE_FACTOR = 0.75 // 色強調係数
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

	// 描画用シフトレジスタ・ラッチ
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
	isSpriteZeroOnLine bool
	nmi                bool

	frame uint64
}

// MARK: PPUのコンストラクタ
func NewPPU(mapper mappers.Mapper, canvas *Canvas) PPU {
	return PPU{
		control:            NewControlRegister(),
		mask:               NewMaskRegister(),
		status:             NewStatusRegister(),
		t:                  NewAddressRegister(),
		v:                  NewAddressRegister(),
		x:                  NewXRegister(),
		w:                  NewWRegister(),
		mapper:             mapper,
		canvas:             canvas,
		dot:                0,
		scanline:           0,
		oamAddress:         0x00,
		dataBuffer:         0x00,
		spriteCount:        0,
		isSpriteZeroOnLine: false,
		nmi:                false,
		frame:              0,
	}
}

// MARK: PPUクロックの更新
func (p *PPU) Tick(cycles uint) {
	for range cycles {
		// 描画設定を取得
		isRenderingEnabled := p.mask.backgroundEnable || p.mask.spriteEnable

		// スキャンライン位置によって処理
		switch {
		case SCANLINE_START <= p.scanline && p.scanline < SCANLINE_POSTRENDER:
			p.tickVisibleScanline(isRenderingEnabled)
		case p.scanline == SCANLINE_VBLANK:
			p.tickVBlankScanline()
		case p.scanline == SCANLINE_PRERENDER:
			p.tickPreRenderScanline(isRenderingEnabled)
		}

		// サイクル, スキャンラインを進める
		p.incrementCycles()
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
	p.status.SetVBlank(false) // 読み取りでVBlankフラグとラッチがクリアされる
	p.w.reset()
	return status
}

// MARK: OAMデータの読み取り (CPU: $2004)
func (p *PPU) ReadOAMData() uint8 {
	return p.oam[p.oamAddress]
}

// MARK: PPUデータの読み取り (CPU: $2007)
func (p *PPU) ReadPPUData() uint8 {
	address := p.v.ToWord() & PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング

	// PPU DATA レジスタの読み出しによってVRAMアドレスは自動的にインクリメントされる
	p.incrementVRAMAddress()

	// CPUからの読み取りは内部バッファにより一回分遅延する
	value := p.dataBuffer
	p.dataBuffer = p.ReadPPUMemory(address)

	// パレットテーブルのみ遅延無しで読み取り
	if 0x3F00 <= address && address <= 0x3FFF {
		value = p.ReadPPUMemory(address)

		// パレットテーブルの読み出し時はネームテーブルのミラーリング領域のデータが内部バッファにセットされる
		p.dataBuffer = p.ReadPPUMemory(address & PPU_NAMETABLE_ADDRESS_MASK)
	}

	return value
}

// MARK: PPUメモリマップの読み取り
func (p *PPU) ReadPPUMemory(address uint16) uint8 {
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
	case PPU_ADDRESS_START <= address && address <= 0x1FFF: // パターンテーブル (CHR ROM)
		return p.mapper.ReadCharacterROM(address)
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル (VRAM)
		vramAddress := p.mirrorVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		return p.vram[vramAddress]
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE

		// スプライトパレットNの0番目の色は背景パレットNの0番目がミラーリングされる
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10
		}
		return p.paletteTable[paletteTableIndex]
	default:
		return 0x00
	}
}

// MARK: PPUコントロールレジスタの書き込み (CPU: $2000)
func (p *PPU) WritePPUControl(value uint8) {
	prev := p.control.GenerateNMI()

	p.control.SetFromByte(value)
	p.t.updateNameTable(value)

	// VBlank中のNMIフラグ操作
	if !prev && p.control.GenerateNMI() && p.status.VBlank() {
		// GenerateNMIがセットされたタイミングでNMIフラグは即時セットされる
		p.nmi = true
	} else if prev && !p.control.GenerateNMI() {
		// GenerateNMIがクリアされたタイミングでNMIフラグは即時クリアされる
		p.nmi = false
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
	// 1回目はXレジスタも更新 (fineX)
	if !p.w.latch {
		p.x.update(value)
	}

	p.t.updateScroll(value, p.w.latch) // Tレジスタは毎回更新 (fineY / coarseX / coarseY)
	p.w.toggle()
}

// MARK: PPUアドレスの書き込み (CPU: $2006)
func (p *PPU) WritePPUAddress(value uint8) {
	p.t.updateAddress(value, p.w.latch)

	// 2回目の書き込み時はTレジスタをVレジスタにコピー
	if p.w.latch {
		p.t.copyAllBitsTo(&p.v)
	}

	p.w.toggle()
}

// MARK: PPUデータの書き込み (CPU: $2007)
func (p *PPU) WritePPUData(value uint8) {
	address := p.v.ToWord() & PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()
	p.WritePPUMemory(address, value)
}

// MARK: PPUデータの書き込み (CPU: $2007)
func (p *PPU) WritePPUMemory(address uint16, value uint8) {
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
	case PPU_ADDRESS_START <= address && address <= 0x1FFF: // パターンテーブル (CHR RAM)
		if p.mapper.IsCharacterRAM() {
			p.mapper.WriteCharacterRAM(address, value)
		}
	case 0x2000 <= address && address <= 0x3EFF: // ネームテーブル
		vramAddress := p.mirrorVRAMAddress(address & PPU_NAMETABLE_ADDRESS_MASK)
		p.vram[vramAddress] = value
	case 0x3F00 <= address && address <= 0x3FFF: // パレットテーブル
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE

		// スプライトパレットNの0番目の色は背景パレットNの0番目がミラーリングされる
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10
		}
		p.paletteTable[paletteTableIndex] = value
	default:
	}
}

// MARK: DMA転送の実行 (CPU: $4014)
func (p *PPU) DMATransfer(bytes *[256]uint8) {
	for _, value := range *bytes {
		p.oam[p.oamAddress] = value
		p.oamAddress++
	}
}

// MARK: 可視スキャンラインの処理
func (p *PPU) tickVisibleScanline(isRenderingEnabled bool) {
	// レンダリングが無効な場合は背景色を出力
	if !isRenderingEnabled {
		if 1 <= p.dot && p.dot <= 256 {
			p.renderBackdropPixel()
		}
		return
	}

	// ラインの先頭でセカンダリOAMを初期化
	if p.dot == 1 {
		p.clearSecondaryOAM()
	}

	// スプライトの評価
	if p.dot == 256 {
		p.evaluateNextLineSprite()
	}

	// 描画
	if 1 <= p.dot && p.dot <= 256 {
		p.renderPixel()
	}

	// 描画用シフトレジスタのシフト
	p.shiftRegisters()

	// 背景・スプライトフェッチ
	p.fetchBackgroundPipeline()
	p.fetchSpritePipeline()

	// スクロールの更新
	p.updateCounters()
}

// MARK: VBlankラインの処理
func (p *PPU) tickVBlankScanline() {
	if p.dot == 1 {
		// ラインの先頭でVBlankフラグを立てる
		p.status.SetVBlank(true)
		if p.control.GenerateNMI() {
			p.nmi = true
		}
	}
}

// MARK: プリレンダーラインの処理
func (p *PPU) tickPreRenderScanline(isRenderingEnabled bool) {
	// ラインの先頭各種フラグのクリア
	if p.dot == 1 {
		p.status.SetVBlank(false)
		p.status.SetSpriteZeroHit(false)
		p.status.SetSpriteOverflow(false)
	}

	// レンダリングが無効な場合は何もしない
	if !isRenderingEnabled {
		return
	}

	// セカンダリOAMの初期化
	if p.dot == 1 {
		p.clearSecondaryOAM()
	}

	// スプライト評価
	if p.dot == 256 {
		p.evaluateNextLineSprite()
	}

	// 描画用シフトレジスタのシフト
	p.shiftRegisters()

	// 背景・スプライトフェッチ
	p.fetchBackgroundPipeline()
	p.fetchSpritePipeline()

	// スクロールの更新
	p.updateCounters()
}

// MARK: 次のスキャンラインで描画するスプライトの評価
func (p *PPU) evaluateNextLineSprite() {
	p.spriteCount = 0 // 評価中のライン上のスプライトの出現数
	spriteHeight := p.control.SpriteSize()

	// 次のスキャンラインを計算，プリレンダーライン以降は次フレームの0ライン目になる
	nextScanline := p.scanline + 1
	if nextScanline > SCANLINE_PRERENDER {
		nextScanline = 0
	}

	// プライマリOAMを順番に評価
	for i := range len(p.oam) / OAM_SPRITE_SIZE {
		primaryBase := uint(i) * OAM_SPRITE_SIZE
		spriteY := uint(p.oam[primaryBase+OAM_SPRITE_Y_POS]) + 1 // OAMのY座標は表示座標-1のため補正

		// 次のラインに重なっているか判定
		if nextScanline < spriteY || nextScanline >= spriteY+uint(spriteHeight) {
			continue
		}

		// 重なっているスプライトをプライマリOAMからセカンダリOAMへロード
		if p.spriteCount < MAX_SPRITE_COUNT {
			// プライマリOAMの先頭であればスプライト0として位置を記憶
			if i == 0 {
				p.isSpriteZeroOnLine = true
			}

			// セカンダリOAMにプライマリOAMの中身をコピー
			secondaryBase := p.spriteCount * OAM_SPRITE_SIZE
			p.secondaryOAM[secondaryBase+OAM_SPRITE_Y_POS] = p.oam[primaryBase+OAM_SPRITE_Y_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_TILE_POS] = p.oam[primaryBase+OAM_SPRITE_TILE_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_ATTR_POS] = p.oam[primaryBase+OAM_SPRITE_ATTR_POS]
			p.secondaryOAM[secondaryBase+OAM_SPRITE_X_POS] = p.oam[primaryBase+OAM_SPRITE_X_POS]

			// ライン上のスプライト数をインクリメント
			p.spriteCount++
		} else {
			// 9個目が存在し，描画が無効でない場合スプライトオーバーフローフラグをセットし評価を止める
			if p.mask.backgroundEnable || p.mask.spriteEnable {
				p.status.SetSpriteOverflow(true)
				break
			}
		}
	}
}

// MARK: 背景フェッチパイプライン
func (p *PPU) fetchBackgroundPipeline() {
	if (1 <= p.dot && p.dot <= 256) || (321 <= p.dot && p.dot <= 336) {
		// 次ラインの背景パターンをフェッチ
		p.fetchBackground()
	}
}

// MARK: スプライトフェッチパイプライン
func (p *PPU) fetchSpritePipeline() {
	if 257 <= p.dot && p.dot <= 320 {
		// 次ラインのスプライトパターンをフェッチ
		spriteIndex := (p.dot - 257) / TILE_SIZE
		p.fetchSprite(spriteIndex)
	}
}

// MARK: 背景ネームテーブル/属性テーブルのフェッチ
func (p *PPU) fetchBackground() {
	/*
		背景フェッチタイミング (8ドット毎)

		| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | ...
		-----------------------------------------
		| - |  NT   |  AT   | BG lo | BG hi | ...

		1 ~ 7ドット目のフェッチはラッチに保存され，8ドット目でラッチからシフタにまとめて反映される
	*/

	switch p.dot % TILE_SIZE {
	case 1: // ネームテーブルのフェッチ
		nameTableAddress := p.getNameTableAddress()
		tileIndex := p.ReadPPUMemory(nameTableAddress)
		p.backgroundLatch.tileIndex = tileIndex
	case 3: // 属性テーブルのフェッチ
		attributeTableAddress := p.getAttributeTableAddress()
		attribute := p.ReadPPUMemory(attributeTableAddress)
		p.backgroundLatch.attribute = attribute
	case 5: // パターンテーブル(下位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(p.backgroundLatch.tileIndex, false)
		pattern := p.ReadPPUMemory(patternTableAddress)
		p.backgroundLatch.patternLower = pattern
	case 7: // パターンテーブル(上位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(p.backgroundLatch.tileIndex, true)
		pattern := p.ReadPPUMemory(patternTableAddress)
		p.backgroundLatch.patternUpper = pattern
	case 0: // ラッチからシフトレジスタへロード
		p.backgroundShift.load(&p.backgroundLatch, p.v.coarseX, p.v.coarseY)
		p.v.incrementHorizontal() // VRAMアドレスの水平インクリメント
	}
}

// MARK: スプライトネームテーブルのフェッチ
func (p *PPU) fetchSprite(index uint) {
	/*
		スプライトフェッチタイミング (8ドット毎)

		| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
		-----------------------------------
		| - |   -   |   -   | Splow | SpHi  |
	*/

	// 現在のラインで使用されないシフトレジスタをリセット
	if p.spriteCount <= index {
		p.spriteShifts[index].reset()
		return
	}

	switch p.dot % TILE_SIZE {
	case 5: // パターンテーブル(下位)のフェッチ
		address := p.getSpritePatternAddress(index, false)
		p.spriteLatch.patternLower = p.ReadPPUMemory(address)
	case 7: // パターンテーブル(上位)のフェッチ
		address := p.getSpritePatternAddress(index, true)
		p.spriteLatch.patternUpper = p.ReadPPUMemory(address)
	case 0: // ラッチ・セカンダリOAMからシフトレジスタへロード
		basePtr := index * OAM_SPRITE_SIZE

		spriteX := p.secondaryOAM[basePtr+OAM_SPRITE_X_POS]
		attributes := p.secondaryOAM[basePtr+OAM_SPRITE_ATTR_POS]
		flipH := (attributes>>6)&1 == 1

		p.spriteShifts[index].load(&p.spriteLatch, flipH)
		p.spriteShifts[index].attributes = attributes
		p.spriteShifts[index].xDistance = spriteX
		p.spriteShifts[index].isSpriteZero = (index == 0 && p.isSpriteZeroOnLine)
	}
}

// MARK: シフトレジスタのシフト処理
func (p *PPU) shiftRegisters() {
	// 背景のシフト
	if (1 <= p.dot && p.dot <= 256) || (321 <= p.dot && p.dot <= 336) {
		p.backgroundShift.shift()
	}

	// スプライトのシフト
	if 1 <= p.dot && p.dot <= 256 {
		for i := range p.spriteShifts {
			if p.spriteShifts[i].xDistance > 0 {
				// 描画位置までは描画位置カウンタを減算
				p.spriteShifts[i].xDistance--
			} else {
				// 描画位置になったら描画するドットを送り始める
				p.spriteShifts[i].shift()
			}
		}
	}
}

// MARK: スクロールカウンタ等の更新処理
func (p *PPU) updateCounters() {
	if p.dot == 256 {
		p.v.incrementVertical()
	}
	if p.dot == 257 {
		p.t.copyHorizontalBitsTo(&p.v)
		p.oamAddress = 0x00
	}
	if p.scanline == SCANLINE_PRERENDER {
		if 280 <= p.dot && p.dot <= 304 {
			p.t.copyVerticalBitsTo(&p.v)
		}
	}
}

// MARK: 背景色(Backdrop)の取得
func (p *PPU) getBackdropColor() sdl.Color {
	address := p.v.ToWord() & PPU_MEMORY_ADDRESS_MASK
	if !p.mask.backgroundEnable && !p.mask.spriteEnable && 0x3F00 <= address && address <= 0x3FFF {
		paletteTableIndex := (address - 0x3F00) % PPU_PALETTE_TABLE_SIZE
		if paletteTableIndex >= 0x10 && paletteTableIndex%4 == 0 {
			paletteTableIndex -= 0x10 // $3F10, $3F14, $3F18, $3F1C は $3F00 番台にミラーされる
		}
		return PALETTE[p.paletteTable[paletteTableIndex]]
	}
	return PALETTE[p.paletteTable[0x00]]
}

// MARK: 背景色(Backdrop)の描画
func (p *PPU) renderBackdropPixel() {
	screenX := p.dot - 1
	screenY := p.scanline
	color := p.getBackdropColor()
	p.canvas.SetPixel(screenX, screenY, p.getEmphasizedColor(color))
}

// MARK: ピクセルの書き込み
func (p *PPU) renderPixel() {
	screenX := p.dot - 1
	screenY := p.scanline

	// 現在のドットの背景・スプライトパターン，透過状態を取得
	bgPattern, bgAttribute := p.getBackgroundPixel()
	spPattern, spAttributes, isSpriteZero, spPriority := p.getSpritePixel()

	bgOpaque := (bgPattern != 0)
	spOpaque := (spPattern != 0)

	// スプライト0ヒット判定
	if isSpriteZero && bgOpaque && spOpaque && screenX < SCREEN_WIDTH-1 {
		// 左端タイルのマスクがされている時は左端ではスプライト0ヒットが起こらない
		leftMasked := !p.mask.leftmostBackgroundEnable || !p.mask.leftmostSpriteEnable
		if !(leftMasked && screenX < TILE_SIZE) {
			p.status.SetSpriteZeroHit(true)
		}
	}

	// 優先順位に基づいて色を決定
	var color sdl.Color
	switch {
	case !bgOpaque && !spOpaque: // 両方透明
		color = p.getBackdropColor()
	case !bgOpaque && spOpaque: // 背景のみ透明
		color = p.getSpriteColor(spAttributes, spPattern)
	case bgOpaque && !spOpaque: // スプライトのみ透明
		color = p.getBackgroundColor(bgAttribute, bgPattern)
	case bgOpaque && spOpaque: // 両方不透明
		if spPriority == 0 {
			// スプライト優先の場合
			color = p.getSpriteColor(spAttributes, spPattern)
		} else {
			// 背景優先の場合
			color = p.getBackgroundColor(bgAttribute, bgPattern)
		}
	}

	// 決定した色を描画
	p.canvas.SetPixel(screenX, screenY, p.getEmphasizedColor(color))
}

// MARK: VRAMアドレスをミラーリング
func (p *PPU) mirrorVRAMAddress(address uint16) uint16 {
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
	address := (p.v.ToWord() + step)
	p.v.SetFromWord(address & PPU_MEMORY_ADDRESS_MASK)
}

// MARK: サイクルを進める
func (p *PPU) incrementCycles() {
	// ドットを進める
	p.dot++

	// 端に達したらドットを0に戻しスキャンラインを進める
	if p.dot > SCANLINE_END {
		p.dot = 0
		p.scanline++

		// プリレンダーラインに達したらスキャンラインを0に戻しフレーム数を進める
		if p.scanline > SCANLINE_PRERENDER {
			p.scanline = 0
			p.frame++
		}
	}
}

// MARK: セカンダリOAMのクリア
func (p *PPU) clearSecondaryOAM() {
	for i := range p.secondaryOAM {
		p.secondaryOAM[i] = 0xFF
	}
	p.isSpriteZeroOnLine = false
}

// MARK: Vレジスタから描画中の行・列のネームテーブルのアドレスを取得
func (p *PPU) getNameTableAddress() uint16 {
	/*
		基準となるネームテーブルの始点 ($2000 | $2400 | $2800 | $2C00) と
		Vレジスタの下位12ビット(NN YYYYY XXXXX)の論理和で現在の行・列のアドレスが求まる
	*/
	return 0x2000 | (p.v.ToWord() & 0x0FFF)
}

// MARK: Vレジスタから描画中の属性テーブルのアドレスを取得
func (p *PPU) getAttributeTableAddress() uint16 {
	/*
		ベースアドレス: $23C0 (ネームテーブルの末尾)
		p.v & 0x0C00: どの画面（ネームテーブル）かを選択
		(v >> 4) & 0x38: Y座標 (行) をメタタイル単位(4行ごと)に変換
		(v >> 2) & 0x07: X座標 (列) をメタタイル単位(4列ごと)に変換
	*/

	v := p.v.ToWord()
	base := 0x23C0 | (v & 0x0C00)
	row := uint16(p.v.coarseY >> 2)
	column := uint16(p.v.coarseX >> 2)
	offset := row*8 + column
	return base | offset
}

// MARK: Vレジスタから描画中の背景タイルのパターンテーブルのアドレスを取得
func (p *PPU) getBackgroundPatternAddress(tileIndex uint8, isUpper bool) uint16 {
	base := p.control.BackgroundPatternTableAddress()
	offset := 0
	if isUpper {
		offset = TILE_SIZE
	}
	return base + (uint16(tileIndex) * TILE_SIZE * 2) + uint16(p.v.fineY) + uint16(offset)
}

// MARK: 描画中のスプライトタイルのパターンテーブルのアドレスを取得
func (p *PPU) getSpritePatternAddress(index uint, isUpper bool) uint16 {
	// スプライトの情報を取得
	secondaryBase := index * OAM_SPRITE_SIZE
	spriteY := p.secondaryOAM[secondaryBase+OAM_SPRITE_Y_POS] + 1 // OAMのY座標は表示座標-1のため補正
	tileIndex := p.secondaryOAM[secondaryBase+OAM_SPRITE_TILE_POS]
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

	// タイル内の行数
	tileY := uint16(targetScanline - uint(spriteY))

	// 垂直反転の場合はタイル内行数も反転
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
		base = uint16(tileIndex&0x01) * 0x1000 // タイル番号のビット0がネームテーブル選択
		tileIndex = tileIndex & 0xFE
		if tileY >= TILE_SIZE {
			tileIndex++
			tileY -= TILE_SIZE
		}
	}

	return base + (uint16(tileIndex) * TILE_SIZE * 2) + tileY + uint16(offset)
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
			isSpriteZero = p.spriteShifts[i].isSpriteZero
			priority = (attributes >> 5) & 0x01
			return pixel, attributes, isSpriteZero, priority
		}
	}
	return 0x00, 0x00, false, 0x00
}

// MARK: 属性情報とピクセルの値から色を取得するメソッド
func (p *PPU) getBackgroundColor(attribute uint8, pattern uint8) sdl.Color {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		colorIndex := p.paletteTable[0x00]
		return PALETTE[colorIndex]
	}

	paletteTableIndex := ((attribute & 0x03) << 2) + pattern
	colorIndex := p.paletteTable[paletteTableIndex]
	return PALETTE[colorIndex]
}

// MARK: 属性情報とピクセルの値からスプライトの色を取得するメソッド
func (p *PPU) getSpriteColor(attributes uint8, pattern uint8) sdl.Color {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		colorIndex := p.paletteTable[0x00]
		return PALETTE[colorIndex]
	}

	// スプライトパレットの開始位置 0x10 をオフセットとして加算
	paletteTableIndex := 0x10 + ((attributes & 0x03) << 2) + pattern
	colorIndex := p.paletteTable[paletteTableIndex]
	return PALETTE[colorIndex]
}

// MARK: マスクレジスタの値から色強調を反映した色を取得するメソッド
func (p *PPU) getEmphasizedColor(baseColor sdl.Color) sdl.Color {
	// 全色強調ビットがクリアであれば元の色を返す
	if !p.mask.emphasizeRed && !p.mask.emphasizeGreen && !p.mask.emphasizeBlue {
		return baseColor
	}

	r := baseColor.R
	g := baseColor.G
	b := baseColor.B

	// 協調されていない色成分を弱める
	if !p.mask.emphasizeRed {
		r = uint8(float32(r) * COLOR_EMPHASIZE_FACTOR)
	}
	if !p.mask.emphasizeGreen {
		g = uint8(float32(g) * COLOR_EMPHASIZE_FACTOR)
	}
	if !p.mask.emphasizeBlue {
		b = uint8(float32(b) * COLOR_EMPHASIZE_FACTOR)
	}

	return sdl.Color{R: r, G: g, B: b}
}

// MARK: 待機中のNMI状態をチェックするメソッド
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

// MARK: 経過フレーム数を取得するメソッド
func (p *PPU) Frame() uint64 {
	return p.frame
}
