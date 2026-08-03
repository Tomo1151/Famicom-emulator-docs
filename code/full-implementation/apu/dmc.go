package apu

// MARK: 変数定義
var (
	DMC_PITCH_TABLE = [16]uint16{
		0x1AC, 0x17C, 0x154, 0x140, 0x11E, 0x0FE, 0x0E2, 0x0D6,
		0x0BE, 0x0A0, 0x08E, 0x080, 0x06A, 0x054, 0x048, 0x036,
	}
)

// DeltaModulationChannelの定義
type DeltaModulationChannel struct {
	register      DMCRegister
	readMemory    MemoryReader
	shiftRegister DMCShiftRegisteer

	timer         uint16 // タイマ
	timerPeriod   uint16 // チャンネルの周期
	deltaCounter  uint8  // デルタカウンタ初期値
	sampleAddress uint16 // サンプル開始アドレス
	bytesLeft     uint   // サンプル長
	irqEnabled    bool   // IRQ有効フラグ
	loop          bool   // ループフラグ

	irq    bool
	output float32 // 出力値
}

// MARK: DeltaModulationChannelのコンストラクタ
func NewDeltaModulationChannel() *DeltaModulationChannel {
	return &DeltaModulationChannel{
		register:      NewDMCRegister(),
		timer:         0x0000,
		timerPeriod:   0x0000,
		deltaCounter:  0x00,
		sampleAddress: 0xC000,
		bytesLeft:     0x0000,
		irqEnabled:    false,
		loop:          false,
		output:        0.0,
	}
}

// MARK: DMCのクロック
func (dmc *DeltaModulationChannel) Tick() {
	// クロック毎にタイマを進める
	if dmc.timer > 0 {
		dmc.timer--
	} else {
		// タイマのリセット
		dmc.timer = dmc.timerPeriod

		// シフトレジスタが空になった時
		if dmc.shiftRegister.isEmpty() {
			// まだ再生するサンプルが残っていたら
			if dmc.bytesLeft > 0 {
				// シフトレジスタをリセット
				dmc.shiftRegister.reset()

				// 次のサンプルをフェッチ
				dmc.shiftRegister.SetValue(dmc.readMemory(dmc.sampleAddress))

				// サンプルアドレスを進めて残りのバイト数をデクリメント
				dmc.sampleAddress++
				dmc.bytesLeft--

				// サンプルアドレスのオーバーフローを丸める
				if dmc.sampleAddress == 0 {
					dmc.sampleAddress = 0x8000
				}

				// 最後まで再生したとき
				if dmc.bytesLeft == 0 {
					// ループフラグが有効であればリロード
					if dmc.register.loop {
						dmc.reload()
					} else if dmc.register.irqEnabled {
						dmc.irq = true
					}
				}
			}
		} else {
			// シフトレジスタからデルタカウンタへ反映
			if dmc.shiftRegister.Value()&0x01 != 0 {
				if dmc.deltaCounter < 0x7F {
					dmc.deltaCounter += 2
				}
			} else {
				if dmc.deltaCounter > 1 {
					dmc.deltaCounter -= 2
				}
			}

			// シフトレジスタのシフト
			dmc.shiftRegister.shift()
		}
	}

	// デルタカウンタの値を出力値としてセット
	dmc.output = float32(dmc.deltaCounter)
}

// MARK: DMCの書き込み
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
		dmc.irqEnabled = (value&0x80 != 0)
		dmc.loop = (value&0x40 != 0)
		dmc.timerPeriod = DMC_PITCH_TABLE[(value & 0x0F)]
	case 0x4011:
		/*
			$4011 書き込み
			- デルタカウンタ初期値
		*/
		dmc.deltaCounter = (value & 0x7F)
	case 0x4012:
		/*
			$4012 書き込み
			- サンプルアドレス
		*/
		dmc.sampleAddress = 0xC000 + uint16(value)<<6
	case 0x4013:
		/*
			$4013 書き込み
			- サンプル長
		*/
		dmc.bytesLeft = uint(value)<<4 + 1
	}
}

// MARK: DMCの出力
func (dmc *DeltaModulationChannel) Output() float32 {
	return dmc.output
}

// MARK: IRQの取得
func (dmc *DeltaModulationChannel) PollIRQ() bool {
	if dmc.irq {
		dmc.irq = false
		return true
	}
	return false
}

// MARK: 状態のリロード
func (dmc *DeltaModulationChannel) reload() {
	// レジスタからセットされた時のデータをリロード
	dmc.sampleAddress = 0xC000 + (uint16(dmc.register.sampleAddress) << 4)
	dmc.bytesLeft = (uint(dmc.register.sampleLength) << 4) + 1
}

// MARK: MemoryReaderのセット
func (dmc *DeltaModulationChannel) SetMemoryReader(reader MemoryReader) {
	dmc.readMemory = reader
}
