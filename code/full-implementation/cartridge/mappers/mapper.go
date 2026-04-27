package mappers

// MARK: 定数定義
const (
	ROM_DATA_DIR  = "../../rom/"
	SAVE_DATA_DIR = ROM_DATA_DIR + "saves/"
)

const (
	// ページサイズ
	PRG_ROM_PAGE_SIZE uint = 16 * 1024 // 16kB
	PRG_RAM_PAGE_SIZE uint = 8 * 1024  // 16kB
	CHR_ROM_PAGE_SIZE uint = 8 * 1024  // 8kB

	// CPUバス
	PRG_RAM_START uint16 = 0x6000
	PRG_ROM_START uint16 = 0x8000
	PRG_ROM_END   uint16 = 0xFFFF

	// PPUバス
	CHR_ROM_START uint16 = 0x0000
	CHR_ROM_END   uint16 = 0x1FFF
)

const (
	SCANLINE_POSTRENDER = 240
)

const (
	INES_HEADER_SIZE  uint = 16  // iNESフォーマットのヘッダサイズ
	INES_TRAINER_SIZE uint = 512 // iNESフォーマットのトレーナサイズ

	INES_PRG_ROM_PAGE_COUNT_POS uint = 4 // PRG ROMのページ数
	INES_CHR_ROM_PAGE_COUNT_POS uint = 5 // CHR ROMのページ数
	INES_CONTROL_BYTE_1_POS     uint = 6 // コントロールフラグ1のビット位置
	INES_CONTROL_BYTE_2_POS     uint = 7 // コントロールフラグ2のビット位置
	INES_PRG_RAM_UNIT_SIZE_POS  uint = 8 // PRG RAMのサイズ

	/*
		コントロールフラグ 1

		7 6 5 4 3 2 1 0 ビット
		L + + | | | | |
					| | | | +- ミラーリング指定: 0 → 水平 / 1 → 垂直
					| | | +--- バックアップRAMの有無: 0 → 無し / 1 → $6000 ~ $7FFF にマッピング
					| | +----- トレーナ領域の有無: 0 → 無し / 1 → $7000 ~ $71FF にマッピング
					| +------- 4画面ミラーリング指定: 0 → 無効 / 1 → 有効
					+--------- マッパ番号の下位4ビット

		---------------------

		コントロールフラグ 2

		7 6 5 4 3 2 1 0 ビット
		L + + | L | | |
					|   | | +- VS Unisystemフラグ: 0 → 水平 / 1 → 垂直
					|   | +--- PlayChoice-10フラグ: 0 → 無し / 1 → $6000 ~ $7FFF にマッピング
					|   +----- NES2.0フラグ: 0b00 → iNES1.0 / 0b10 → NES2.0フォーマットを使用
					|
					+--------- マッパ番号の上位4ビット
	*/
)

const (
	MIRRORING_VERTICAL Mirroring = iota
	MIRRORING_HORIZONTAL
	MIRRORING_FOURSCREEN
)

// MARK: Mirroringの定義
type Mirroring uint8

// MARK: Mapperの定義
type Mapper interface {
	Init(name string, romdata []uint8, savedata []uint8)

	ReadProgramRAM(address uint16) uint8
	ReadProgramROM(address uint16) uint8
	ReadCharacterROM(address uint16) uint8
	WriteProgramRAM(address uint16, value uint8)
	WriteProgramROM(address uint16, value uint8)
	WriteCharacterRAM(address uint16, value uint8)

	Mirroring() Mirroring
	IsCharacterRAM() bool
	ProgramROM() []uint8
	CharacterROM() []uint8

	GenerateScanlineIRQ(scanline uint, isRenderingEnabled bool)
	IRQ() bool
	Save()

	MapperInfo() string
}

// MARK: 初期ミラーリングの取得
func InitMirroring(romdata []uint8) (mirroring Mirroring) {
	isFourScreen := (romdata[INES_CONTROL_BYTE_1_POS] & 0b1000) != 0
	isVertical := (romdata[INES_CONTROL_BYTE_1_POS] & 0b0001) != 0

	if isFourScreen {
		mirroring = MIRRORING_FOURSCREEN
	} else if isVertical {
		mirroring = MIRRORING_VERTICAL
	} else {
		mirroring = MIRRORING_HORIZONTAL
	}

	return mirroring
}

// MARK: ROMファイルからPRG ROMとCHR ROMを抽出する関数
func ExtractROMs(romdata []uint8) (programROM, characterROM []uint8) {
	programStart := programROMStartAddress(romdata)
	programSize := programROMSize(romdata)
	characterStart := characterROMStartAddress(romdata)
	characterSize := characterROMSize(romdata)

	programROM = romdata[programStart:(programStart + programSize)]
	characterROM = romdata[characterStart:(characterStart + characterSize)]

	return programROM, characterROM
}

// MARK: ROMファイルからROG RAMとCHR RAMを生成する関数
func GenerateRAMs(romdata []uint8) (programRAM, characterRAM []uint8) {
	programRAM = make([]uint8, max(programRAMSize(romdata), PRG_RAM_PAGE_SIZE))
	characterRAM = make([]uint8, CHR_ROM_PAGE_SIZE)
	return programRAM, characterRAM
}

// MARK: PRG ROMの先頭アドレスを取得する関数
func programROMStartAddress(romdata []uint8) (address uint) {
	skipTrainer := (romdata[INES_CONTROL_BYTE_1_POS] & 0b100) != 0
	var offset uint
	if skipTrainer {
		offset = INES_TRAINER_SIZE
	} else {
		offset = 0
	}

	return INES_HEADER_SIZE + offset // ヘッダ + トレーナ後のアドレスを返却
}

// MARK: CHR ROMの先頭アドレスを取得する関数
func characterROMStartAddress(romdata []uint8) (address uint) {
	// PRG ROMの開始アドレスにサイズを足してCHR ROMの先頭アドレスを求める
	startAddress := programROMStartAddress(romdata)
	romSize := programROMSize(romdata)
	return startAddress + romSize
}

// MARK: PRG ROMのサイズを取得する関数
func programROMSize(romdata []uint8) (size uint) {
	return uint(romdata[INES_PRG_ROM_PAGE_COUNT_POS]) * PRG_ROM_PAGE_SIZE
}

// MARK: PRG RAMのサイズを取得する関数
func programRAMSize(romdata []uint8) (size uint) {
	return uint(romdata[INES_PRG_RAM_UNIT_SIZE_POS]) * PRG_RAM_PAGE_SIZE
}

// MARK: CHR ROMのサイズを取得する関数
func characterROMSize(romdata []uint8) (size uint) {
	return uint(romdata[INES_CHR_ROM_PAGE_COUNT_POS]) * CHR_ROM_PAGE_SIZE
}
