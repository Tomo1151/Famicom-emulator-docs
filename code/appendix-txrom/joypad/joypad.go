package joypad

// MARK: 定数定義
const (
	JOYPAD_BUTTON_A_POS JoyPadButton = iota
	JOYPAD_BUTTON_B_POS
	JOYPAD_BUTTON_SELECT_POS
	JOYPAD_BUTTON_START_POS
	JOYPAD_BUTTON_UP_POS
	JOYPAD_BUTTON_DOWN_POS
	JOYPAD_BUTTON_LEFT_POS
	JOYPAD_BUTTON_RIGHT_POS
)

// MARK: JoyPadButtonの定義
type JoyPadButton uint8

// MARK: JoyPadの定義
type JoyPad struct {
	/*
		ボタン状態 (state)

		7 6 5 4 3 2 1 0 ビット
		------- -------
		R L D U s S B A
		| | | | | | | |
		| | | | | | | +- A: Aボタン
		| | | | | | +--- B: Bボタン
		| | | | | +----- S: セレクトボタン
		| | | | +------- s: スタートボタン
		| | | +--------- U: 矢印ボタン上
		| | +----------- D: 矢印ボタン下
		| +------------- L: 矢印ボタン左
		+--------------- R: 矢印ボタン右
	*/

	strobe bool  // ストローブ信号
	index  uint8 // 読み取り位置
	state  uint8 // ボタンの状態
	latch  uint8 // 内部ラッチのシフトレジスタ
}

// MARK: JoyPadのコンストラクタ
func NewJoypad() JoyPad {
	return JoyPad{
		strobe: false,
		index:  0x00,
		state:  0x00,
		latch:  0x00,
	}
}

// MARK: コントローラ状態の読み取り
func (j *JoyPad) ReadJoyPad() uint8 {
	/*
		ボタン状態読み取り

		7 6 5 4 3 2 1 0 ビット
		------- -------
		- - - T W M E B
		L + | | | | | |
		    | | | | | +- B: 標準コントローラボタン
		    | | | | +--- E: 拡張ポートコントローラ
		    | | | +----- M: 2Pマイク
		    | | +------- W: 光線銃光検知
		    | +--------- T: 光線銃トリガー
		    +----------- -: オープンバス
	*/
	const openBus = 0x40

	// ストローブ信号がHIGHの時は常にAボタンの状態を返す
	if j.strobe {
		return openBus | (j.state & 0x01)
	}

	/*
		ボタン読み取り順
		A → B → セレクト → スタート → 上 → 下 → 左 → 右

		$4016/4017がReadされるごとにシフトすることで読み取り順通りに値を返す
		8回目以降はAボタンが返される
	*/
	var bit uint8
	if j.index < 8 {
		bit = (j.latch >> j.index) & 0x01
	} else {
		bit = 0x01
	}
	j.index++

	return openBus | bit
}

// MARK: コントローラ状態の書き込み
func (j *JoyPad) WriteJoyPad(value uint8) {
	/*
		コントローラ書き込み

		7 6 5 4 3 2 1 0 ビット
		------- -------
		- - - - - E E B
		          L | |
		            | +- B: 標準コントローララッチ
		            +--- E: 拡張ポートラッチ (未対応)
	*/
	prev := j.strobe
	j.strobe = ((value & 0x01) == 1)

	// ストローブ信号がHIGHの時は常にボタン状態が反映，読み取りボタン位置がリセットされ続ける
	if j.strobe {
		j.index = 0x00
		j.latch = j.state
		return
	}

	// ストローブ信号のダウンエッジでボタン状態・読み取りボタン位置を確定
	if prev && !j.strobe {
		j.index = 0x00
		j.latch = j.state
	}
}

// MARK: コントローラボタン状態のセット
func (j *JoyPad) SetButtonState(index JoyPadButton, pressed bool) {
	if pressed {
		j.state |= (1 << index) // 押された位置のビットをセット
	} else {
		j.state &^= (1 << index) // 押された位置のビットをクリア
	}
}
