package apu

// MARK: DMC再生レートテーブル
var (
	DMC_PITCH_TABLE = [16]uint16{
		0x1AC, 0x17C, 0x154, 0x140,
		0x11E, 0x0FE, 0x0E2, 0x0D6,
		0x0BE, 0x0A0, 0x08E, 0x080,
		0x06A, 0x054, 0x048, 0x036,
	}
)

// MARK: DeltaModulationChannelの定義
type DeltaModulationChannel struct {
	register      DMCRegister
	readMemory    MemoryReader
	shiftRegister DMCShiftRegisteer

	silence           bool   // 無音フラグ
	sampleBuffer      uint8  // サンプルバッファ
	sampleBufferEmpty bool   // サンプルバッファの状態
	timer             uint16 // タイマ
	timerPeriod       uint16 // チャンネルの周期
	deltaCounter      uint8  // デルタカウンタ初期値
	sampleAddress     uint16 // サンプル開始アドレス
	bytesLeft         uint   // 残りサンプル数
	irqEnabled        bool   // IRQ有効フラグ
	loop              bool   // ループフラグ

	irq    bool    // 割り込み待機状態
	output float32 // 出力値
}

// MARK: DeltaModulationChannelのコンストラクタ
func NewDeltaModulationChannel() *DeltaModulationChannel {
	return &DeltaModulationChannel{
		register:          NewDMCRegister(),
		shiftRegister:     NewDMCShiftRegister(),
		silence:           true,
		sampleBuffer:      0,
		sampleBufferEmpty: true,
		timer:             0,
		timerPeriod:       0,
		deltaCounter:      0,
		sampleAddress:     0xC000,
		bytesLeft:         0,
		irqEnabled:        false,
		loop:              false,
		irq:               false,
		output:            0,
	}
}

// MARK: DMCのクロック
func (dmc *DeltaModulationChannel) Tick() {
	// バッファにデータがあり，シフトレジスタが空なら移動
	dmc.reloadShiftRegister()

	// バッファが空ならメモリから1バイト読み込み
	dmc.tickMemoryReader()

	// クロック毎にタイマを進める
	if dmc.timer > 0 {
		dmc.timer--
		return
	}

	// 分周器の励起時にタイマをリロード
	if dmc.timerPeriod > 0 {
		dmc.timer = dmc.timerPeriod - 1
	}

	// 1ビット出力
	dmc.tickOutputUnit()
	dmc.output = float32(dmc.deltaCounter)
}

// MARK: Output Unitのクロック
func (dmc *DeltaModulationChannel) tickOutputUnit() {
	// シフトレジスタが空のときは何もしない
	if dmc.shiftRegister.isEmpty() {
		return
	}

	if !dmc.silence {
		// サンプルバッファが空でない場合
		bit := dmc.shiftRegister.shift()

		if bit != 0 {
			if dmc.deltaCounter <= 125 {
				dmc.deltaCounter += 2
			}
		} else {
			if dmc.deltaCounter >= 2 {
				dmc.deltaCounter -= 2
			}
		}
	} else {
		// サンプルバッファが空の場合
		dmc.shiftRegister.shift()
	}

	// シフトレジスタが空になったら即座にサンプルのフェッチを行う
	if dmc.shiftRegister.isEmpty() {
		dmc.reloadShiftRegister()
		dmc.tickMemoryReader()
	}
}

// MARK: Shift Registerへのロード
func (dmc *DeltaModulationChannel) reloadShiftRegister() {
	// シフトレジスタが空でない場合は何もしない
	if !dmc.shiftRegister.isEmpty() {
		return
	}

	// サンプルバッファが空の時は無音フラグをセットする
	if dmc.sampleBufferEmpty {
		dmc.silence = true
		return
	}

	// サンプルバッファからシフトレジスタにデータをロード
	dmc.shiftRegister.reset()
	dmc.shiftRegister.SetValue(dmc.sampleBuffer)

	// サンプルバッファを空にし，無音フラグをクリア
	dmc.sampleBufferEmpty = true
	dmc.silence = false

	// サンプルのフェッチ
	dmc.tickMemoryReader()
}

// MARK: Memory Readerのクロック
func (dmc *DeltaModulationChannel) tickMemoryReader() {
	// サンプルバッファ，未再生のサンプルが残っている場合は何もしない
	if !dmc.sampleBufferEmpty || dmc.bytesLeft == 0 {
		return
	}

	// サンプルアドレスから1バイト分フェッチして空フラグをクリア
	if dmc.readMemory != nil {
		dmc.sampleBuffer = dmc.readMemory(dmc.sampleAddress)
		dmc.sampleBufferEmpty = false
	}

	// フェッチ後にサンプルアドレスをインクリメントし，残りサンプル数をデクリメント
	dmc.sampleAddress++
	if dmc.sampleAddress == 0 {
		dmc.sampleAddress = 0x8000 // オーバーフロー時には最初に戻す
	}
	dmc.bytesLeft--

	// 残りサンプルが無くなったとき
	if dmc.bytesLeft == 0 {
		if dmc.loop {
			// ループフラグが有効であればリロード
			dmc.reload()
		} else if dmc.irqEnabled {
			// IRQが有効の場合はIRQを発生させる
			dmc.irq = true
		}
	}
}

// MARK: DMCのリロード
func (dmc *DeltaModulationChannel) reload() {
	// サンプルアドレスとサンプル長をレジスタからロード
	dmc.sampleAddress = 0xC000 + (uint16(dmc.register.sampleAddress) << 6)
	dmc.bytesLeft = (uint(dmc.register.sampleLength) << 4) + 1
}

// MARK: DMC有効/無効 ($4015書き込み時)
func (dmc *DeltaModulationChannel) SetEnabled(enabled bool) {
	if enabled {
		// 有効にするとき，再生が終わっていたらリロードとフェッチを行う
		if dmc.bytesLeft == 0 {
			dmc.reload()
			dmc.tickMemoryReader()
		}
	} else {
		// 無効にするとき，再生を終えて全てをクリア
		dmc.bytesLeft = 0
		dmc.sampleBufferEmpty = true
		dmc.shiftRegister.reset()
	}

	// ステータスレジスタ書き込みでIRQはクリアされる
	dmc.irq = false
}

// MARK: DMCレジスタ書き込み
func (dmc *DeltaModulationChannel) write(address uint16, value uint8) {
	dmc.register.write(address, value)

	switch address {
	case 0x4010:
		/*
			$4010 書き込み
			- IRQフラグ
			- ループフラグ
			- 再生レート
		*/
		dmc.irqEnabled = (value & 0x80) != 0
		dmc.loop = (value & 0x40) != 0
		dmc.timerPeriod = DMC_PITCH_TABLE[value&0x0F]

		// IRQ無効化時には待機中のIRQもクリアされる
		if !dmc.irqEnabled {
			dmc.irq = false
		}
	case 0x4011:
		/*
			$4011 書き込み
			- デルタカウンタ初期値
		*/
		dmc.deltaCounter = value & 0x7F
	case 0x4012:
		/*
			$4012 書き込み
			- サンプルアドレス

			※ 初期値をレジスタのみにロード
			　 リロードのタイミングでセットされる
		*/
	case 0x4013:
		/*
			$4013 書き込み
			- サンプル長

			※ 初期値をレジスタのみにロード
			　 リロードのタイミングでセットされる
		*/
	}
}

// MARK: DMC出力の取得
func (dmc *DeltaModulationChannel) Output() float32 {
	return dmc.output
}

// MARK: IRQの取得
func (dmc *DeltaModulationChannel) IRQ() bool {
	return dmc.irq
}

// MARK: IRQのセット
func (dmc *DeltaModulationChannel) SetIRQ(value bool) {
	dmc.irq = value
}

// MARK: 再生中フラグの取得
func (dmc *DeltaModulationChannel) IsActive() bool {
	return dmc.bytesLeft > 0 || !dmc.sampleBufferEmpty
}

// MARK: Memory Readerのセット
func (dmc *DeltaModulationChannel) SetMemoryReader(reader MemoryReader) {
	dmc.readMemory = reader
}
