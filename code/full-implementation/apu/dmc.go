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

// MARK: DMCチャンネル
type DeltaModulationChannel struct {
	register   DMCRegister
	readMemory MemoryReader

	// Output Unit
	shiftRegister DMCShiftRegisteer
	silence       bool

	// Sample Buffer
	sampleBuffer      uint8
	sampleBufferEmpty bool

	// Timer
	timer       uint16
	timerPeriod uint16

	// Delta Counter
	deltaCounter uint8

	// Memory Reader
	sampleAddress uint16
	bytesLeft     uint

	// Control
	irqEnabled bool
	loop       bool

	// IRQ
	irq bool

	// Output
	output float32
}

// MARK: コンストラクタ
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

// MARK: DMCクロック
func (dmc *DeltaModulationChannel) Tick() {
	// 1. バッファにデータがあり、シフトレジスタが空なら移動
	dmc.reloadShiftRegister()

	// 2. バッファが空ならメモリから1バイト読み込み
	dmc.clockMemoryReader()

	// タイマ進行
	if dmc.timer > 0 {
		dmc.timer--
		return
	}

	// タイマ再ロード
	if dmc.timerPeriod > 0 {
		dmc.timer = dmc.timerPeriod - 1
	}

	// 3. 1bit出力（DAC更新）
	dmc.clockOutputUnit()

	dmc.output = float32(dmc.deltaCounter)
}

// MARK: Output Unitのクロック
func (dmc *DeltaModulationChannel) clockOutputUnit() {
	if dmc.shiftRegister.isEmpty() {
		return
	}

	if !dmc.silence {
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
		dmc.shiftRegister.shift()
	}

	// シフトレジスタが空になったら即座に次の充填を試みる
	if dmc.shiftRegister.isEmpty() {
		dmc.reloadShiftRegister()
		dmc.clockMemoryReader()
	}
}

// MARK: Shift Registerへのロード
func (dmc *DeltaModulationChannel) reloadShiftRegister() {
	if !dmc.shiftRegister.isEmpty() {
		return
	}

	if dmc.sampleBufferEmpty {
		dmc.silence = true
		return
	}

	// Sample Buffer -> Shift Register
	dmc.shiftRegister.reset()
	dmc.shiftRegister.SetValue(dmc.sampleBuffer)

	dmc.sampleBufferEmpty = true
	dmc.silence = false

	dmc.clockMemoryReader()
}

// MARK: Memory Readerのクロック
func (dmc *DeltaModulationChannel) clockMemoryReader() {
	if !dmc.sampleBufferEmpty || dmc.bytesLeft == 0 {
		return
	}

	if dmc.readMemory != nil {
		dmc.sampleBuffer = dmc.readMemory(dmc.sampleAddress)
		dmc.sampleBufferEmpty = false
	}

	dmc.sampleAddress++
	if dmc.sampleAddress == 0 {
		dmc.sampleAddress = 0x8000
	}

	dmc.bytesLeft--

	if dmc.bytesLeft == 0 {
		if dmc.loop {
			dmc.reload()
		} else if dmc.irqEnabled {
			dmc.irq = true
		}
	}
}

// MARK: DMC再ロード
func (dmc *DeltaModulationChannel) reload() {
	dmc.sampleAddress = 0xC000 + (uint16(dmc.register.sampleAddress) << 6)
	dmc.bytesLeft = (uint(dmc.register.sampleLength) << 4) + 1
}

// MARK: DMC有効/無効 ($4015書き込み時)
func (dmc *DeltaModulationChannel) SetEnabled(enabled bool) {
	if enabled {
		if dmc.bytesLeft == 0 {
			dmc.reload()
			dmc.clockMemoryReader()
		}
	} else {
		dmc.bytesLeft = 0
		dmc.sampleBufferEmpty = true
		dmc.shiftRegister.reset()
	}
	dmc.irq = false
}

// MARK: DMCレジスタ書き込み
func (dmc *DeltaModulationChannel) write(address uint16, value uint8) {
	dmc.register.write(address, value)

	switch address {
	case 0x4010:
		dmc.irqEnabled = (value & 0x80) != 0
		dmc.loop = (value & 0x40) != 0
		dmc.timerPeriod = DMC_PITCH_TABLE[value&0x0F]

		if !dmc.irqEnabled {
			dmc.irq = false
		}

	case 0x4011:
		dmc.deltaCounter = value & 0x7F

	case 0x4012:
		// Sample Address (register内部保持)

	case 0x4013:
		// Sample Length (register内部保持)
	}
}

// MARK: DMC出力
func (dmc *DeltaModulationChannel) Output() float32 {
	return dmc.output
}

// MARK: IRQ取得
func (dmc *DeltaModulationChannel) IRQ() bool {
	return dmc.irq
}

// MARK: IRQ設定
func (dmc *DeltaModulationChannel) SetIRQ(value bool) {
	dmc.irq = value
}

// MARK: 再生中か ($4015 Bit 4 読出用)
func (dmc *DeltaModulationChannel) IsActive() bool {
	return dmc.bytesLeft > 0 || !dmc.sampleBufferEmpty
}

// MARK: Memory Reader設定
func (dmc *DeltaModulationChannel) SetMemoryReader(reader MemoryReader) {
	dmc.readMemory = reader
}
