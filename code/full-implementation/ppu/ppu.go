package ppu

import (
	"fc-emu/bus"
)

// MARK: 定数定義
const (
	PPU_PRIMARY_OAM_SIZE   = 64 * OAM_SPRITE_SIZE // 64スプライト
	PPU_SECONDARY_OAM_SIZE = 8 * OAM_SPRITE_SIZE  // 8スプライト

	PPU_VRAM_ADDRESS_SPACE = 0x4000

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
	bus          *bus.PPUBus                   // PPU用メモリバス
	oam          [PPU_PRIMARY_OAM_SIZE]uint8   // Object Attribute Memory
	secondaryOAM [PPU_SECONDARY_OAM_SIZE]uint8 // Secondary OAM

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

	canvas *Canvas // 描画キャンバスの参照

	dot             uint
	scanline        uint
	oamAddress      uint8
	dataBuffer      uint8
	spriteCount     uint
	spriteZeroIndex uint
	nmi             bool

	frame uint64
}

// MARK: PPUのコンストラクタ
func NewPPU(bus *bus.PPUBus, canvas *Canvas) PPU {
	return PPU{
		control:         NewControlRegister(),
		mask:            NewMaskRegister(),
		status:          NewStatusRegister(),
		t:               NewAddressRegister(),
		v:               NewAddressRegister(),
		x:               NewXRegister(),
		w:               NewWRegister(),
		bus:             bus,
		canvas:          canvas,
		dot:             0,
		scanline:        0,
		oamAddress:      0x00,
		dataBuffer:      0x00,
		spriteCount:     0,
		spriteZeroIndex: SPRITE_ZERO_NOT_FOUND,
		nmi:             false,
		frame:           0,
	}
}

// MARK: PPUクロックの更新
func (p *PPU) Tick(cycles uint) {
	// 描画設定を取得
	isRenderingEnabled := p.mask.backgroundEnable || p.mask.spriteEnable

	for range cycles {
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
	address := p.v.ToByte() & bus.PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()

	// CPUからの読み取りは内部バッファにより一回分遅延する
	value := p.dataBuffer
	p.dataBuffer = p.bus.ReadByteFrom(address)

	// パレットテーブルのみ遅延無しで読み取り
	if 0x3F00 <= address && address <= 0x3FFF {
		value = p.bus.ReadByteFrom(address)
	}

	return value
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

// MARK: PPUデータの書き込み (CPU: $2007)
func (p *PPU) WritePPUData(value uint8) {
	address := p.v.ToByte() & bus.PPU_MEMORY_ADDRESS_MASK // $4000-$FFFF のミラーリング
	p.incrementVRAMAddress()
	p.bus.WriteByteAt(address, value)
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
	if p.dot == 1 {
		// ラインの先頭でセカンダリOAMを初期化
		p.clearSecondaryOAM()
	}
	if p.dot == 256 {
		p.evaluateNextLineSprite()
	}

	// 描画
	if 1 <= p.dot && p.dot <= 256 {
		p.renderPixel()

		if isRenderingEnabled {
			p.shiftSpriteRegisters()
		}
	}

	if !isRenderingEnabled {
		return
	}

	// 背景フェッチ
	p.fetchBackgroundPipeline()

	// スクロール更新
	if p.dot == 256 {
		p.v.incrementVertical()
	}
	if p.dot == 257 {
		p.t.copyHorizontalBitsTo(&p.v)
	}

	// マッパー割り込みの生成
	if p.dot == 260 {
		p.bus.Mapper().GenerateScanlineIRQ(p.scanline, isRenderingEnabled)
	}

	// スプライトフェッチ
	p.fetchSpritePipeline()
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
	if p.dot == 1 {
		p.status.SetVBlank(false)
		p.status.SetSpriteZeroHit(false)
		p.status.SetSpriteOverflow(false)
		p.clearSecondaryOAM()
	}
	if p.dot == 256 {
		p.evaluateNextLineSprite()
	}

	if !isRenderingEnabled {
		return
	}

	p.fetchBackgroundPipeline()

	if p.dot == 256 {
		p.v.incrementVertical()
	}
	if p.dot == 257 {
		p.t.copyHorizontalBitsTo(&p.v)
	}
	if 280 <= p.dot && p.dot <= 304 {
		p.t.copyVerticalBitsTo(&p.v)
	}

	// スプライトフェッチ
	p.fetchSpritePipeline()
}

// MARK: 次のスキャンラインで描画するスプライトの評価
func (p *PPU) evaluateNextLineSprite() {
	p.spriteCount = 0
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
				p.spriteZeroIndex = p.spriteCount
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
			// 9個目が存在する場合スプライトオーバーフローフラグをセットし評価を止める
			p.status.SetSpriteOverflow(true)
			break
		}
	}
}

// MARK: 背景フェッチパイプライン
func (p *PPU) fetchBackgroundPipeline() {
	if (1 <= p.dot && p.dot <= 256) || (321 <= p.dot && p.dot <= 336) {
		// 背景パターンのフェッチ
		p.backgroundShift.shift()
		p.fetchBackground()

		// 8ドット毎にVレジスタの水平アドレスをインクリメント
		if p.dot%TILE_SIZE == 0 {
			p.v.incrementHorizontal()
		}
	}
}

// MARK: スプライトフェッチパイプライン
func (p *PPU) fetchSpritePipeline() {
	if 257 <= p.dot && p.dot <= 320 {
		// 次のライン用にタイルをフェッチ
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
		tile := p.bus.ReadByteFrom(nameTableAddress)
		p.backgroundLatch.nameTable = tile
	case 3: // 属性テーブルのフェッチ
		attributeTableAddress := p.getAttributeTableAddress()
		attribute := p.bus.ReadByteFrom(attributeTableAddress)
		// Vレジスタの位置に応じて該当する2ビットを抽出する
		shift := ((p.v.coarseY>>1)&1)<<2 | ((p.v.coarseX>>1)&1)<<1
		p.backgroundLatch.attribute = (attribute >> shift) & 0x03
	case 5: // パターンテーブル(下位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(false)
		pattern := p.bus.ReadByteFrom(patternTableAddress)
		p.backgroundLatch.patternLower = pattern
	case 7: // パターンテーブル(上位)のフェッチ
		patternTableAddress := p.getBackgroundPatternAddress(true)
		pattern := p.bus.ReadByteFrom(patternTableAddress)
		p.backgroundLatch.patternUpper = pattern
	case 0: // ラッチからシフトレジスタへロード
		p.backgroundShift.load(&p.backgroundLatch)
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
		p.spriteLatch.patternLower = p.bus.ReadByteFrom(address)
	case 7: // パターンテーブル(上位)のフェッチ
		address := p.getSpritePatternAddress(index, true)
		p.spriteLatch.patternUpper = p.bus.ReadByteFrom(address)
	case 0: // ラッチ・セカンダリOAMからシフトレジスタへロード
		basePtr := index * OAM_SPRITE_SIZE

		spriteX := p.secondaryOAM[basePtr+OAM_SPRITE_X_POS]
		attributes := p.secondaryOAM[basePtr+OAM_SPRITE_ATTR_POS]
		flipH := (attributes>>6)&1 == 1

		p.spriteShifts[index].load(&p.spriteLatch, flipH)
		p.spriteShifts[index].attributes = attributes
		p.spriteShifts[index].xDistance = spriteX
		p.spriteShifts[index].isSpriteZero = (index == p.spriteZeroIndex)
	}
}

// MARK: スプライトシフトレジスタのシフト
func (p *PPU) shiftSpriteRegisters() {
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
	var color rgb
	switch {
	case !bgOpaque && !spOpaque: // 両方透明
		color = PALETTE[p.bus.ReadPalette(0x00)]
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

// MARK: VRAMアドレスのインクリメント
func (p *PPU) incrementVRAMAddress() {
	step := uint16(p.control.VRAMAddressIncrement())
	address := (p.v.ToByte() + step)
	p.v.SetFromWord(address & bus.PPU_MEMORY_ADDRESS_MASK)
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
	p.spriteZeroIndex = SPRITE_ZERO_NOT_FOUND
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
	// スプライトの情報を取得
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
		base = uint16(spriteTile&0x01) * 0x1000 // タイル番号のビット0がネームテーブル選択
		spriteTile = spriteTile & 0xFE
		if tileY >= TILE_SIZE {
			spriteTile++
			tileY -= TILE_SIZE
		}
	}

	return base + (uint16(spriteTile) * TILE_SIZE * 2) + tileY + uint16(offset)
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
func (p *PPU) getBackgroundColor(attribute uint8, pattern uint8) rgb {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		paletteIndex := p.bus.ReadPalette(0x00)
		return PALETTE[paletteIndex]
	}

	paletteTableIndex := ((attribute & 0x03) << 2) + pattern
	paletteIndex := p.bus.ReadPalette(paletteTableIndex)
	return PALETTE[paletteIndex]
}

// MARK: 属性情報とピクセルの値からスプライトの色を取得するメソッド
func (p *PPU) getSpriteColor(attributes uint8, pattern uint8) rgb {
	// 透明の場合は背景色を返す
	if pattern == 0x00 {
		paletteIndex := p.bus.ReadPalette(0x00)
		return PALETTE[paletteIndex]
	}

	// スプライトパレットの開始位置 0x10 をオフセットとして加算
	paletteTableIndex := 0x10 + ((attributes & 0x03) << 2) + pattern
	paletteIndex := p.bus.ReadPalette(paletteTableIndex)
	return PALETTE[paletteIndex]
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
