package apu

// MARK: 変数定義
var (
	// 長さカウンタのルックアップテーブル
	LENGTH_COUNTER_TABLE = [32]uint8{
		0x0A, 0xFE, 0x14, 0x02, 0x28, 0x04, 0x50, 0x06,
		0xA0, 0x08, 0x3C, 0x0A, 0x0E, 0x0C, 0x1A, 0x0E,
		0x0C, 0x10, 0x18, 0x12, 0x30, 0x14, 0x60, 0x16,
		0xC0, 0x18, 0x48, 0x1A, 0x10, 0x1C, 0x20, 0x1E,
	}
)

// MARK: Envelopeの定義
type Envelope struct {
	counter uint8 // エンベロープのカウンタ値
	timer   uint8 // 分周器のタイマ値
	period  uint8 // 周期
	enabled bool  // エンベロープ有効フラグ
	loop    bool  // エンベロープループフラグ
}

// MARK: Envelopeのコンストラクタ
func NewEnvelope() Envelope {
	return Envelope{
		counter: 0x0F,
		timer:   0x01,
		period:  0x00,
		enabled: false,
		loop:    false,
	}
}

// MARK: エンベロープのクロック
func (e *Envelope) Tick() {
	// クロック毎に分周器のタイマを進める
	e.timer--

	// 分周器が励起するまでは何もしない
	if e.timer != 0 {
		return
	}

	if e.counter != 0 {
		// もしもカウンタが0でなければデクリメント
		e.counter--
	} else {
		// カウンタが0かつループフラグがセットされているならカウンタを0x0Fに
		if e.loop {
			e.reset()
		}
	}

	// 分周器の周期はエンベロープ周期+1にセットされる
	e.timer = e.period + 1
}

// MARK: エンベロープのリセット
func (e *Envelope) reset() {
	e.counter = 0x0F
	e.timer = e.period + 1
}

// MARK: エンベロープの更新
func (e *Envelope) update(period uint8, enabled, loop bool) {
	e.period = period
	e.enabled = enabled
	e.loop = loop
}

// MARK: エンベロープからボリュームを取得する
func (e *Envelope) Volume() float32 {
	if e.enabled {
		// エンベロープが有効の場合は現在のカウンタ値を返す
		return float32(e.counter)
	} else {
		// エンベロープが無効の場合はエンベロープ周期をそのまま返す(最大値を使用)
		return float32(e.period)
	}
}

// MARK: SweepUnitの定義
type SweepUnit struct {
	timerPeriod uint16 // チャンネルの周期
	timer       uint8  // スイープユニットのカウンタ値
	reload      bool   // リロードフラグ
	shift       uint8  // スイープ量
	direction   uint8  // スイープ方向
	period      uint8  // 分周器の周期
	enabled     bool   // スイープユニットの有効フラグ
}

// MARK: SweepUnitのコンストラクタ
func NewSweepUnit() SweepUnit {
	return SweepUnit{
		timerPeriod: 0x0000,
		timer:       0x00,
		reload:      false,
		shift:       0x00,
		direction:   0x00,
		period:      0x00,
		enabled:     false,
	}
}

// MARK: スイープユニットのクロック
func (su *SweepUnit) Tick(lengthCounter *LengthCounter, is1ch bool) {
	// クロック毎に分周器のタイマを進める
	su.timer--

	// リロードフラグがクリア，分周器が励起するまでの間は何もしない
	if !su.reload && su.timer > 0 {
		return
	}

	/*
		以下の3つの条件にすべて当てはまるときのみチャンネルの周期を更新する
		- スイープ有効フラグがセット
		- スイープ量が0でない
		- チャンネルの長さカウンタが0でない
	*/
	if !su.Enabled() || su.shift == 0 || lengthCounter.Muted() {
		return
	}

	// 更新する周波数を計算
	var target uint16
	diff := su.timerPeriod >> uint16(su.shift)

	if su.direction == 0 {
		// 下向きのスイープ
		target = su.timerPeriod + diff
	} else {
		// 上向きのスイープ
		if is1ch {
			// 1chでは1の補数を使用 (NOT)
			target = su.timerPeriod - (diff + 1)
		} else {
			// 2chでは2の補数を使用 (NEG)
			target = su.timerPeriod - diff
		}
	}

	// スイープ後の周期に値をセットし，リロードフラグをクリア
	su.timerPeriod = target
	su.reload = false

	// 分周器の周期はスイープ周期+1にセットされる
	su.timer = su.period + 1
}

// MARK: スイープユニットの更新
func (su *SweepUnit) update(shift, direction, period uint8, enabled bool) {
	su.shift = shift
	su.direction = direction
	su.period = period
	su.enabled = enabled

	// 更新時にリロードフラグをセット
	su.reload = true
}

// MARK: スイープユニットからチャンネル周期の取得
func (su *SweepUnit) Period() uint16 {
	return su.timerPeriod
}

// MARK: スイープユニットのミュート状態の取得
func (su *SweepUnit) Muted() bool {
	// チャンネルの周期が8未満または0x07FF以上の場合，スイープを停止し，チャンネルを無音化する
	return su.timerPeriod < 8 || 0x07FF < su.timerPeriod
}

// MARK: スイープユニットの有効/無効の取得
func (su *SweepUnit) Enabled() bool {
	return su.enabled
}

// MARK: LinearCounterの定義
type LinearCounter struct {
	counter     uint8 // 線形カウンタのカウンタ値
	timerReload uint8 // カウンタのリロード値
	control     bool  // コントロールフラグ
	relaod      bool  // リロードフラグ
}

// MARK: LinearCounterのコンストラクタ
func NewLinearCounter() LinearCounter {
	return LinearCounter{
		counter:     0x00,
		timerReload: 0x00,
		control:     false,
		relaod:      false,
	}
}

// MARK: 線形カウンタのクロック
func (lc *LinearCounter) Tick() {
	if lc.relaod {
		// リロードフラグがセットの場合，即座にカウンタ値をリロード値に更新
		lc.counter = lc.timerReload
	} else if lc.counter > 0 {
		// リロードフラグがクリアかつカウンタ値が0でなければカウンタ値をデクリメント
		lc.counter--
	}

	// コントロールフラグがクリアであればリロードフラグもクリア
	if !lc.control {
		lc.relaod = false
	}
}

// MARK: 線形カウンタのリロードフラグをセット
func (lc *LinearCounter) SetReload() {
	lc.relaod = true
}

// MARK: 線形カウンタの更新メソッド
func (lc *LinearCounter) update(timerReload uint8, control bool) {
	lc.timerReload = timerReload
	lc.control = control
}

// MARK: 線形カウンタのミュート状態の取得
func (lc *LinearCounter) Muted() bool {
	return lc.counter == 0
}

// MARK: LengthCounterの定義
type LengthCounter struct {
	counter uint8 // カウンタ値
	halt    bool  // 停止フラグ
}

// MARK: LengthCounterのコンストラクタ
func NewLengthCounter() LengthCounter {
	return LengthCounter{
		counter: 0x00,
		halt:    false,
	}
}

// MARK: 長さカウンタのクロック
func (lc *LengthCounter) Tick() {
	// 停止フラグがクリアかつカウンタが0でないならカウンタをデクリメント
	if !lc.halt && lc.counter > 0 {
		lc.counter--
	}
}

// MARK: 長さカウンタの更新
func (lc *LengthCounter) load(index uint8) {
	// ルックアップテーブルからカウンタをセット
	lc.counter = LENGTH_COUNTER_TABLE[index]
}

// MARK: 長さカウンタの強制クリア
func (lc *LengthCounter) clear() {
	lc.counter = 0
}

// MARK: 長さカウンタのミュート状態の取得
func (lc *LengthCounter) Muted() bool {
	return lc.counter == 0
}

// MARK: 長さカウンタの停止フラグのセット
func (lc *LengthCounter) SetHalt(halt bool) {
	lc.halt = halt
}
