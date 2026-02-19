package mappers

// MARK: NROMの定義
type NROM struct {
	name           string
	mirroring      Mirroring
	isCharacterRAM bool
	programROM     []uint8
	characterROM   []uint8
}

// MARK: NROMのコンストラクタ
func (r *NROM) Init(name string, romdata []uint8, savedata []uint8) {
	programROM, characetrROM := ExtractROMs(romdata)

	r.name = name
	r.mirroring = InitMirroring(romdata)
	r.programROM = programROM
	r.characterROM = characetrROM
}

// MARK: プログラムRAMの読み取りメソッド
func (r *NROM) ReadProgramRAM(address uint16) uint8 {
	// NOTE: NROMにはプログラムRAMがないため，オープンバスの挙動としてアドレスの上位バイトを返す
	return uint8(address >> 8)
}

// MARK: プログラムROMの読み取りメソッド
func (r *NROM) ReadProgramROM(address uint16) uint8 {
	// ROMは$8000からマッピングされているため，オフセット分引いて配列のインデックスにする
	romAddress := address - 0x8000

	// 16kBのROMでアドレスが16kB以上の場合はミラーリング
	if len(r.programROM) == 0x4000 && romAddress >= 0x4000 {
		romAddress %= 0x4000
	}
	return r.programROM[romAddress]
}

// MARK: キャラクタROMの読み取りメソッド
func (r *NROM) ReadCharacterROM(address uint16) uint8 {
	return r.characterROM[address]
}

// MARK: プログラムRAMの書き込みメソッド
func (r *NROM) WriteProgramRAM(address uint16, value uint8) {
	// NOTE: NROMにはプログラムRAM無し
}

// MARK: プログラムROMの書き込みメソッド
func (r *NROM) WriteProgramROM(address uint16, value uint8) {
	// NOTE: NROMはプログラムROMへの書き込みでは何も起こらない
}

// MARK: キャラクタROMの書き込みメソッド
func (r *NROM) WriteCharacterRAM(address uint16, value uint8) {
	if !r.isCharacterRAM || int(address) < 0 || int(address) >= len(r.characterROM) {
		return
	}
	r.characterROM[address] = value
}

// MARK: ミラーリングを取得するメソッド
func (r *NROM) Mirroring() Mirroring {
	return r.mirroring
}

// MARK: キャラクタRAMの有無を取得するメソッド
func (r *NROM) IsCharacterRAM() bool {
	return r.isCharacterRAM
}

// MARK: プログラムROMを取得するメソッド
func (r *NROM) ProgramROM() []uint8 {
	return r.programROM
}

// MARK: キャラクタROMを取得するメソッド
func (r *NROM) CharacterROM() []uint8 {
	return r.characterROM
}

// MARK: NROMの情報を取得するメソッド
func (r *NROM) MapperInfo() string {
	return "NROM (Mapper 000)"
}
