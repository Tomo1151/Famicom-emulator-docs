package mappers

import (
	"fmt"
	"os"
)

// MARK: 定数定義
const (
	TXROM_PRG_ROM_BANK_SIZE = 8 * 1024 // 8kB
	TXROM_PRG_RAM_BANK_SIZE = 8 * 1024 // 8kB
	TXROM_CHR_ROM_BANK_SIZE = 1 * 1024 // 1kB
)

// MARK: TxROMの定義
type TxROM struct {
	name           string
	mirroring      Mirroring
	isCharacterRAM bool

	bankSelect uint8
	bankData   [8]uint8
	ramProtect uint8
	irqLatch   uint8
	irqReload  bool
	irqEnabled bool
	irqCounter uint8
	irq        bool

	programROM   []uint8
	programRAM   []uint8
	characterROM []uint8
	characterRAM []uint8
}

// MARK: TxROMのコンストラクタ
func (r *TxROM) Init(name string, romdata []uint8, savedata []uint8) {
	r.name = name
	programROM, characterROM := ExtractROMs(romdata)
	programRAM, characterRAM := GenerateRAMs(romdata)

	r.mirroring = InitMirroring(romdata)
	r.isCharacterRAM = (len(characterROM) == 0)

	r.bankSelect = 0
	r.ramProtect = 0x00
	r.irqLatch = 0x00
	r.irqReload = false
	r.irqEnabled = false

	r.programROM = programROM
	r.programRAM = programRAM
	r.characterROM = characterROM
	r.characterRAM = characterRAM

	// バンクデータの初期化
	for i := range r.bankData {
		r.bankData[i] = 0x00
	}

	// プログラムRAMの初期化
	for i := range r.programRAM {
		r.programRAM[i] = 0xFF
	}

	// セーブデータのロード
	if len(savedata) != 0 {
		copy(r.programRAM[:], savedata)
		r.ramProtect = 0x80
	}
}

// MARK: プログラムRAMの読み取りメソッド
func (r *TxROM) ReadProgramRAM(address uint16) uint8 {
	if r.ramProtect&0x80 != 0 {
		// RAM有効ビットが立っている場合のみRAMから読み取り
		ramAddress := address - PRG_RAM_START
		return r.programRAM[ramAddress]
	} else {
		// 無効なRAMアクセスの場合は0xFFを返す
		return 0xFF
	}
}

// MARK: プログラムROMの読み取りメソッド
func (r *TxROM) ReadProgramROM(address uint16) uint8 {
	/*
		バンク選択

		モード0
		$8000-$9FFF:  R6
		$A000-$BFFF:  R7
		$C000-$DFFF: (-2)
		$E000-$FFFF: (-1)

		モード1
		$8000-$9FFF: (-2)
		$A000-$BFFF:  R7
		$C000-$DFFF:  R6
		$E000-$FFFF: (-1)

	*/
	romAddress := r.calcProgramROMAddress(address)
	return r.programROM[romAddress]
}

// MARK: キャラクタROMの読み取りメソッド
func (r *TxROM) ReadCharacterROM(address uint16) uint8 {
	if r.isCharacterRAM {
		return r.characterRAM[r.calcCharacterAddress(address)]
	} else {
		return r.characterROM[r.calcCharacterAddress(address)]
	}
}

// MARK: プログラムRAMの書き込みメソッド
func (r *TxROM) WriteProgramRAM(address uint16, value uint8) {
	// RAM保護が無効な場合のみ書き込む
	if r.ramProtect&0x80 != 0 && r.ramProtect&0x40 == 0 {
		ramAddress := address - PRG_RAM_START
		r.programRAM[ramAddress] = value
	}
}

// MARK: プログラムROMの書き込みメソッド
func (r *TxROM) WriteProgramROM(address uint16, value uint8) {
	/*
		プログラムROM書き込み

		偶数アドレス
		$8000-$9FFE: バンク選択
		$A000-$BFFE: ミラーリング選択
		$C000-$DFFE: IRQラッチ
		$E000-$FFFE: IRQ無効化

		奇数アドレス
		$8001-$9FFF: バンクデータ
		$A001-$BFFF: プログラムRAM保護
		$C001-$DFFF: IRQリロード
		$E001-$FFFF: IRQ有効化
	*/
	isEvenAddress := address&0x01 == 0

	switch {
	case PRG_ROM_START <= address && address <= 0x9FFF:
		if isEvenAddress {
			// バンク選択
			r.bankSelect = value
		} else {
			// バンクデータ
			current := uint(r.bankSelect & 0x07)
			r.bankData[current] = value
		}
	case 0xA000 <= address && address <= 0xBFFF:
		if isEvenAddress {
			// ミラーリング選択
			if value&0x01 == 0 {
				// 下位1ビットが0であれば垂直ミラーリング
				r.mirroring = MIRRORING_VERTICAL
			} else {
				// 下位1ビットが1であれば水平ミラーリング
				r.mirroring = MIRRORING_HORIZONTAL
			}
		} else {
			// プログラムRAM保護
			r.ramProtect = value
		}
	case 0xC000 <= address && address <= 0xDFFF:
		if isEvenAddress {
			// IRQラッチ
			r.irqLatch = value
			r.irqCounter = value
		} else {
			// IRQリロード
			r.irqReload = true
			r.irqCounter = 0
		}
	case 0xE000 <= address && address <= PRG_ROM_END:
		if isEvenAddress {
			// IRQ無効化
			r.irqEnabled = false
			r.irq = false
		} else {
			r.irqEnabled = true
		}
	}
}

// MARK: キャラクタROMの書き込みメソッド
func (r *TxROM) WriteCharacterRAM(address uint16, value uint8) {
	if r.isCharacterRAM {
		r.characterRAM[r.calcCharacterAddress(address)] = value
	}
}

// MARK: スキャンラインによってIRQを発生させるメソッド
func (r *TxROM) GenerateScanlineIRQ(scanline uint, isRenderingEnabled bool) {
	// 可視領域の描画中
	if scanline <= SCANLINE_POSTRENDER && isRenderingEnabled {
		if r.irqReload || r.irqCounter == 0 {
			// リロードフラグがセットまたはカウンタが0ならカウンタをラッチの値でリロード
			r.irqCounter = r.irqLatch
			r.irqReload = false
		} else {
			// そうでなければカウンタをデクリメント
			r.irqCounter--
		}

		// カウンタが0かつIRQが有効ならIRQを発生させる
		if r.irqCounter == 0 && r.irqEnabled {
			r.irq = true
		}
	}
}

// MARK: 待機中のIRQ状態をチェックするメソッド
func (r *TxROM) PollIRQ() bool {
	if r.irq {
		r.irq = false
		return true
	} else {
		return false
	}
}

// MARK: セーブデータの書き出し
func (r *TxROM) Save() {
	// RAM書き込みが有効な場合のみセーブ
	if r.ramProtect&0x80 != 0 {
		filename := SAVE_DATA_DIR + r.name + ".save"
		err := os.WriteFile(filename, r.programRAM[:], 0644)
		if err != nil {
			fmt.Printf("[System] Failed to save game data: %v\n", err)
		} else {
			fmt.Printf("[System] Game data saved to: %s\n", filename)
		}
	}
}

// MARK: プログラムROMアドレスの計算メソッド
func (r *TxROM) calcProgramROMAddress(address uint16) uint {
	/*
		バンク選択

		モード0
		$8000-$9FFF:  R6
		$A000-$BFFF:  R7
		$C000-$DFFF: (-2)
		$E000-$FFFF: (-1)

		モード1
		$8000-$9FFF: (-2)
		$A000-$BFFF:  R7
		$C000-$DFFF:  R6
		$E000-$FFFF: (-1)

	*/

	// バンク選択モード
	mode := (r.bankSelect & 0x40) >> 6

	// バンク番号
	bankMax := uint(len(r.programROM)) / TXROM_PRG_ROM_BANK_SIZE
	lastBank1 := uint(bankMax - 1)
	lastBank2 := uint(bankMax - 2)
	/*
		@NOTE: R6とR7は上位2ビットを無視する (MMC3ではPRG ROMのアドレスバスが6本のため)
		ref: https://www.nesdev.org/wiki/MMC3
	*/
	r6Bank := uint(r.bankData[6]) & 0x3F
	r7Bank := uint(r.bankData[7]) & 0x3F

	var romAddress uint

	// モードによって適切なバンクのデータを返す
	switch mode {
	case 0:
		switch {
		case PRG_ROM_START <= address && address <= 0x9FFF: // R6
			romAddress = uint(address-PRG_ROM_START) + (r6Bank * TXROM_PRG_ROM_BANK_SIZE)
		case 0xA000 <= address && address <= 0xBFFF: // R7
			romAddress = uint(address-0xA000) + (r7Bank * TXROM_PRG_ROM_BANK_SIZE)
		case 0xC000 <= address && address <= 0xDFFF: // (-2)
			romAddress = uint(address-0xC000) + (lastBank2 * TXROM_PRG_ROM_BANK_SIZE)
		case 0xE000 <= address && address <= PRG_ROM_END: // (-1)
			romAddress = uint(address-0xE000) + (lastBank1 * TXROM_PRG_ROM_BANK_SIZE)
		default:
			panic(fmt.Sprintf("Error: unexpected PRG ROM address $%04X", address))
		}
	case 1:
		switch {
		case PRG_ROM_START <= address && address <= 0x9FFF: // (-2)
			romAddress = uint(address-PRG_ROM_START) + (lastBank2 * TXROM_PRG_ROM_BANK_SIZE)
		case 0xA000 <= address && address <= 0xBFFF: // R7
			romAddress = uint(address-0xA000) + (r7Bank * TXROM_PRG_ROM_BANK_SIZE)
		case 0xC000 <= address && address <= 0xDFFF: // R6
			romAddress = uint(address-0xC000) + (r6Bank * TXROM_PRG_ROM_BANK_SIZE)
		case 0xE000 <= address && address <= PRG_ROM_END: // (-1)
			romAddress = uint(address-0xE000) + (lastBank1 * TXROM_PRG_ROM_BANK_SIZE)
		default:
			panic(fmt.Sprintf("Error: unexpected PRG ROM address $%04X", address))
		}
	}

	return romAddress
}

// MARK: キャラクタROM/RAMアドレスの計算メソッド
func (r *TxROM) calcCharacterAddress(address uint16) uint {
	/*
		バンク選択

		モード0
		$0000-$03FF: R0
		$0400-$07FF: R0
		$0800-$0BFF: R1
		$0C00-$0FFF: R1
		$1000-$13FF: R2
		$1400-$17FF: R3
		$1800-$1BFF: R4
		$1C00-$1FFF: R5

		モード1
		$0000-$03FF: R2
		$0400-$07FF: R3
		$0800-$0BFF: R4
		$0C00-$0FFF: R5
		$1000-$13FF: R0
		$1400-$17FF: R0
		$1800-$1BFF: R1
		$1C00-$1FFF: R1

	*/
	mode := (r.bankSelect & 0x80) >> 7

	r0Bank := uint(r.bankData[0] & 0xFE)
	r1Bank := uint(r.bankData[1] & 0xFE)
	r2Bank := uint(r.bankData[2])
	r3Bank := uint(r.bankData[3])
	r4Bank := uint(r.bankData[4])
	r5Bank := uint(r.bankData[5])

	var romAddress uint

	// モードによって適切なバンクのデータを返す
	switch mode {
	case 0:
		switch {
		case CHR_ROM_START <= address && address <= 0x07FF:
			romAddress = uint(address) + (r0Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x0800 <= address && address <= 0x0FFF:
			romAddress = uint(address-0x0800) + (r1Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1000 <= address && address <= 0x13FF:
			romAddress = uint(address-0x1000) + (r2Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1400 <= address && address <= 0x17FF:
			romAddress = uint(address-0x1400) + (r3Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1800 <= address && address <= 0x1BFF:
			romAddress = uint(address-0x1800) + (r4Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1C00 <= address && address <= CHR_ROM_END:
			romAddress = uint(address-0x1C00) + (r5Bank * TXROM_CHR_ROM_BANK_SIZE)
		}
	case 1:
		switch {
		case CHR_ROM_START <= address && address <= 0x03FF:
			romAddress = uint(address) + (r2Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x0400 <= address && address <= 0x07FF:
			romAddress = uint(address-0x0400) + (r3Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x0800 <= address && address <= 0x0BFF:
			romAddress = uint(address-0x0800) + (r4Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x0C00 <= address && address <= 0x0FFF:
			romAddress = uint(address-0x0C00) + (r5Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1000 <= address && address <= 0x17FF:
			romAddress = uint(address-0x1000) + (r0Bank * TXROM_CHR_ROM_BANK_SIZE)
		case 0x1800 <= address && address <= CHR_ROM_END:
			romAddress = uint(address-0x1800) + (r1Bank * TXROM_CHR_ROM_BANK_SIZE)
		}
	}

	return romAddress
}

// MARK: ミラーリングを取得するメソッド
func (r *TxROM) Mirroring() Mirroring {
	return r.mirroring
}

// MARK: IRQを取得するメソッド
func (r *TxROM) IRQ() bool {
	return r.irq
}

// MARK: キャラクタRAMの有無を取得するメソッド
func (r *TxROM) IsCharacterRAM() bool {
	return r.isCharacterRAM
}

// MARK: プログラムROMを取得するメソッド
func (r *TxROM) ProgramROM() []uint8 {
	return r.programROM
}

// MARK: キャラクタROMを取得するメソッド
func (r *TxROM) CharacterROM() []uint8 {
	return r.characterROM
}

// MARK: TxROMの情報を取得するメソッド
func (r *TxROM) MapperInfo() string {
	return "MMC3 TxROM (Mapper 004)"
}
