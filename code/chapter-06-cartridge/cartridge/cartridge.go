package cartridge

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"fc-emu/cartridge/mappers"
)

// MARK: 定数定義
const (
	ROM_DATA_DIR  = "../../rom/"
	SAVE_DATA_DIR = ROM_DATA_DIR + "saves/"
)

// MARK: 変数定義
var (
	NES_TAG = []uint8{0x4E, 0x45, 0x53, 0x1A} // iNESフォーマットのマジックナンバー
)

// MARK: Cartridgeの定義
type Cartridge struct {
	ROM    string // ROMデータのパス
	name   string
	mapper mappers.Mapper
}

// MARK: Cartridgeのコンストラクタ
func NewCartridge(rom string) *Cartridge {
	return &Cartridge{
		ROM: rom,
	}
}

// MARK: ROMファイルの読み込みメソッド
func (c *Cartridge) Load() error {
	ext := filepath.Ext(c.ROM)
	c.name = strings.TrimSuffix(filepath.Base(c.ROM), ext)

	// ROMファイルの読み込み
	romFile, err := os.ReadFile(ROM_DATA_DIR + c.ROM)
	if err != nil {
		return fmt.Errorf("Couldn't load rom file: %s", c.ROM)
	}

	// セーブデータディレクトリの存在確認・作成
	if _, err = os.Stat(SAVE_DATA_DIR); os.IsNotExist(err) {
		os.Mkdir(SAVE_DATA_DIR, 0755)
	}

	// セーブデータファイルの読み込み
	saveFile, err := os.ReadFile(SAVE_DATA_DIR + c.name + ".save")
	if err != nil {
		saveFile = []uint8{}
	}

	// NESタグの検証
	if !reflect.DeepEqual(romFile[0:4], NES_TAG) {
		return fmt.Errorf("Invalid cartridge header: %v", romFile[0:4])
	}

	// iNESヘッダのチェック
	version := (romFile[mappers.INES_CONTROL_BYTE_2_POS] >> 2) & 0b11
	if version != 0 {
		return fmt.Errorf("NES 2.0 format isn't supported")
	}

	// マッパの設定
	mapperNum := (romFile[mappers.INES_CONTROL_BYTE_2_POS] & 0xF0) | (romFile[mappers.INES_CONTROL_BYTE_1_POS] >> 4)
	mapper := c.selectMapper(mapperNum)
	mapper.Init(romFile, saveFile)
	c.mapper = mapper

	// c.DumpInfo(saveFile)
	return nil
}

// MARK: マッパの選択メソッド
func (c *Cartridge) selectMapper(mapperNum uint8) mappers.Mapper {
	switch mapperNum {
	default:
		return &mappers.NROM{}
	}
}

// MARK: Mapperの取得メソッド
func (c *Cartridge) Mapper() mappers.Mapper {
	return c.mapper
}

// MARK: ROM情報の表示メソッド
func (c *Cartridge) DumpInfo(saveFile []uint8) {
	fmt.Printf("Cartridge loaded: %s\n", c.name)
	fmt.Printf("  Mapper      : %s\n", c.mapper.MapperInfo())
	fmt.Printf("  PRG ROM Size: %d bytes\n", len(c.mapper.ProgramROM()))
	fmt.Printf("  CHR ROM Size: %d bytes\n", len(c.mapper.CharacterROM()))
	fmt.Printf("  CHR RAM     : %v\n", c.mapper.IsCharacterRAM())
	if len(saveFile) != 0 {
		fmt.Println("  Save data   : loaded")
	} else {
		fmt.Println("  Save data   : no")
	}
	var mirroring string
	switch c.mapper.Mirroring() {
	case mappers.MIRRORING_FOURSCREEN:
		mirroring = "Four Screen"
	case mappers.MIRRORING_VERTICAL:
		mirroring = "Vertical"
	case mappers.MIRRORING_HORIZONTAL:
		mirroring = "Horizontal"
	default:
		mirroring = "Unknown"
	}
	fmt.Printf("  Mirroring   : %s\n", mirroring)
}
