package apu

// DeltaModulationChannelの定義
type DeltaModulationChannel struct {
	register DMCRegister
	output   float32
}

// MARK: DeltaModulationChannelのコンストラクタ
func NewDeltaModulationChannel() DeltaModulationChannel {
	return DeltaModulationChannel{
		register: NewDMCRegister(),
		output:   0.0,
	}
}

// MARK: DMCのクロック
func (dmc *DeltaModulationChannel) Tick() {}

// MARK: DMCの書き込み
func (dmc *DeltaModulationChannel) write(address uint16, value uint8) {
	dmc.register.write(address, value)
}

// MARK: DMCの出力
func (dmc *DeltaModulationChannel) Output() float32 {
	return dmc.output
}
