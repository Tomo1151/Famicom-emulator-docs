package mappers

// MARK: NROMの定義
type NROM struct {
	mirroring      Mirroring
	isCharacterRAM bool

	programROM   []uint8
	programRAM   []uint8
	characterROM []uint8
	characterRAM []uint8
}

// MARK: NROMのコンストラクタ
func (r *NROM) Init(romdata []uint8) {
	programROM, characterROM := ExtractROMs(romdata)
	programRAM, characterRAM := GenerateRAMs(romdata)

	r.mirroring = InitMirroring(romdata)
	r.isCharacterRAM = (len(characterROM) == 0)

	r.programROM = programROM
	r.programRAM = programRAM
	r.characterROM = characterROM
	r.characterRAM = characterRAM
}

// MARK: プログラムRAMの読み取りメソッド
func (r *NROM) ReadProgramRAM(address uint16) uint8 {
	// NOTE: 公式のNROM基板では必要ないが，拡張されたFamily Basicなどの互換性を保つ
	romAddress := address - 0x6000
	return r.characterRAM[romAddress]
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
	if r.isCharacterRAM {
		return r.characterROM[address]
	} else {
		return r.characterRAM[address]
	}
}

// MARK: プログラムRAMの書き込みメソッド
func (r *NROM) WriteProgramRAM(address uint16, value uint8) {
	// NOTE: 公式のNROM基板では必要ないが，拡張されたFamily Basicなどの互換性を保つ
	romAddress := address - 0x6000
	r.programRAM[romAddress] = value
}

// MARK: プログラムROMの書き込みメソッド
func (r *NROM) WriteProgramROM(address uint16, value uint8) {
	// NOTE: NROMはプログラムROMへの書き込みでは何も起こらない
}

// MARK: キャラクタROMの書き込みメソッド
func (r *NROM) WriteCharacterRAM(address uint16, value uint8) {
	if r.isCharacterRAM {
		r.characterRAM[address] = value
	}
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
