package apu

// MARK: 変数定義
var (
	LENGTH_COUNTER_TABLE = [32]uint8{
		0x0A, 0xFE, 0x14, 0x02, 0x28, 0x04, 0x50, 0x06,
		0xA0, 0x08, 0x3C, 0x0A, 0x0E, 0x0C, 0x1A, 0x0E,
		0x0C, 0x10, 0x18, 0x12, 0x30, 0x14, 0x60, 0x16,
		0xC0, 0x18, 0x48, 0x1A, 0x10, 0x1C, 0x20, 0x1E,
	}
)

// MARK: Envelopeの定義
type Envelope struct {
	counter uint8
	divider uint8
	rate    uint8
	enabled bool
	loop    bool
}

// MARK: Envelopeのコンストラクタ
func NewEnvelope() Envelope {
	return Envelope{
		counter: 0x0F,
		divider: 0x01,
		rate:    0x00,
		enabled: false,
		loop:    false,
	}
}

// MARK: エンベロープのクロック
func (e *Envelope) Tick() {
	e.divider--

	if e.divider != 0 {
		return
	}

	if e.counter != 0 {
		e.counter--
	} else {
		if e.loop {
			e.reset()
		}
	}
	e.divider = e.rate + 1
}

// MARK: エンベロープのリセット
func (e *Envelope) reset() {
	e.counter = 0x0F
	e.divider = e.rate + 1
}

// MARK: エンベロープの更新
func (e *Envelope) update(rate uint8, enabled, loop bool) {
	e.rate = rate
	e.enabled = enabled
	e.loop = loop
}

// MARK: エンベロープからボリュームを取得する
func (e *Envelope) Volume() float32 {
	if e.enabled {
		return float32(e.counter)
	} else {
		return float32(e.rate)
	}
}

// MARK: SweepUnitの定義
type SweepUnit struct {
	frequency  uint16
	counter    uint8
	muted      bool
	reload     bool
	shift      uint8
	direction  uint8
	timerCount uint8
	enabled    bool
}

// MARK: SweepUnitのコンストラクタ
func NewSweepUnit() SweepUnit {
	return SweepUnit{
		frequency:  0x0000,
		counter:    0x00,
		muted:      true,
		reload:     false,
		shift:      0x00,
		direction:  0x00,
		timerCount: 0x00,
		enabled:    false,
	}
}

// MARK: スイープユニットのクロック
func (su *SweepUnit) Tick(lengthCounter *LengthCounter, is1ch bool) {
	if su.counter > 0 {
		su.counter--
	}

	if su.reload || su.counter == 0 {
		su.counter = su.timerCount + 1
		if su.reload {
			su.reload = false
		}
	} else {
		return
	}

	if !su.Enabled() || su.shift == 0 || lengthCounter.Muted() || su.frequency < 8 {
		return
	}

	var target uint16
	diff := su.frequency >> uint16(su.shift)

	if su.direction == 0 {
		target = su.frequency + diff
	} else {
		if is1ch {
			target = su.frequency - (diff + 1)
		} else {
			target = su.frequency - diff
		}
	}

	if target > 0x7FF || target < 8 {
		su.muted = true
		return
	}

	su.frequency = target
	su.muted = false
}

// MARK: スイープユニットのリセット
func (su *SweepUnit) reset() {
	su.counter = 0x00
	su.muted = false
	su.reload = true
}

// MARK: スイープユニットの更新
func (su *SweepUnit) update(shift, direction, period uint8, enabled bool) {
	su.shift = shift
	su.direction = direction
	su.timerCount = period
	su.enabled = enabled
	su.reload = true
}

// MARK: スイープユニットから周波数の取得
func (su *SweepUnit) Frequency() uint16 {
	return su.frequency
}

// MARK: スイープユニットのミュート状態の取得
func (su *SweepUnit) Muted() bool {
	return su.muted
}

// MARK: スイープユニットの有効/無効の取得
func (su *SweepUnit) Enabled() bool {
	return su.enabled
}

// MARK: LinearCounterの定義
type LinearCounter struct {
	counter uint8
	reload  bool
	count   uint8
	enabled bool
}

// MARK: LinearCounterのコンストラクタ
func NewLinearCounter() LinearCounter {
	return LinearCounter{
		counter: 0x00,
		reload:  false,
		count:   0x00,
		enabled: false,
	}
}

// MARK: 線形カウンタのクロック
func (lc *LinearCounter) Tick() {
	if lc.reload {
		lc.counter = lc.count
	} else if lc.counter > 0 {
		lc.counter--
	}

	if !lc.enabled {
		lc.reload = false
	}
}

// MARK: 線形カウンタのリロード
func (lc *LinearCounter) setReload() {
	lc.reload = true
}

// MARK: 線形カウンタの更新メソッド
func (lc *LinearCounter) update(count uint8, enabled bool) {
	lc.count = count
	lc.enabled = enabled
}

// MARK: 線形カウンタのミュート状態の取得
func (lc *LinearCounter) Muted() bool {
	return (lc.counter == 0)
}

// MARK: LengthCounterの定義
type LengthCounter struct {
	counter uint8
	count   uint8
	enabled bool
}

// MARK: LengthCounterのコンストラクタ
func NewLengthCounter() LengthCounter {
	return LengthCounter{
		counter: 0x00,
		count:   0x00,
		enabled: false,
	}
}

// MARK: 長さカウンタのクロック
func (lc *LengthCounter) Tick() {
	if lc.enabled {
		return
	}

	if lc.count > 0 {
		lc.count--
	}
}

// MARK: 長さカウンタのリロード
func (lc *LengthCounter) reload() {
	lc.counter = lc.count
}

// MARK: 長さカウンタの更新
func (lc *LengthCounter) update(count uint8, enabled bool) {
	lc.count = LENGTH_COUNTER_TABLE[count]
	lc.enabled = enabled
}

// MARK: 長さカウンタのミュート状態の取得
func (lc *LengthCounter) Muted() bool {
	return lc.Muted()
}
