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
		irq:           false,
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
		if dmc.timerPeriod > 0 {
			dmc.timer = dmc.timerPeriod - 1
		}

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
					if dmc.loop {
						// fmt.Printf(
						// 	"bytesLeft=%d loop=%v irq=%v\n",
						// 	dmc.bytesLeft,
						// 	dmc.loop,
						// 	dmc.irqEnabled,
						// )
						// fmt.Println("[DMC.Tick] dmc.reload()")
						dmc.reload()
					} else if dmc.irqEnabled {
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
	// fmt.Printf("loop=%v irq=%v\n", dmc.loop, dmc.irqEnabled)
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

		// IRQが無効になったら即座にIRQフラグをクリアする
		if !dmc.irqEnabled {
			dmc.irq = false
		}
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
		// fmt.Printf("$4012 <- %04X\n", value)
	case 0x4013:
		/*
			$4013 書き込み
			- サンプル長
		*/
		// fmt.Printf("$4013 <- %04X\n", value)
	}
}

// MARK: DMCの出力
func (dmc *DeltaModulationChannel) Output() float32 {
	return dmc.output
}

// MARK: 状態のリロード
func (dmc *DeltaModulationChannel) reload() {
	// fmt.Printf("DMC reloaded: bytes left %04d -> %04d\n", dmc.bytesLeft, (uint(dmc.register.sampleLength<<4) + 1))
	// レジスタからセットされた時のデータをリロード
	dmc.sampleAddress = 0xC000 + (uint16(dmc.register.sampleAddress) << 6)
	dmc.bytesLeft = (uint(dmc.register.sampleLength) << 4) + 1
}

// MARK: DMCの再生/停止
func (dmc *DeltaModulationChannel) SetEnabled(enabled bool) {
	if enabled {
		// DMCが有効で，停止している場合は最初から再生開始
		if dmc.bytesLeft == 0 {
			// fmt.Println("[DMC.SetEnabled] dmc.reload()")
			dmc.reload()
		}
	} else {
		// DMCが向こうの場合は停止
		dmc.bytesLeft = 0
	}
}

// MARK: IRQフラグの取得
func (dmc *DeltaModulationChannel) IRQ() bool {
	return dmc.irq
}

// MARK: IRQフラグのセット
func (dmc *DeltaModulationChannel) SetIRQ(value bool) {
	dmc.irq = value
}

// MARK: DMCの再生状態取得
func (dmc *DeltaModulationChannel) IsActive() bool {
	return dmc.bytesLeft > 0
}

// MARK: MemoryReaderのセット
func (dmc *DeltaModulationChannel) SetMemoryReader(reader MemoryReader) {
	dmc.readMemory = reader
}
