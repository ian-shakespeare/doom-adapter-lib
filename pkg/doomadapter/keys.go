package doomadapter

// KeyCode is a doomgeneric key code, as defined in doomkeys.h. Most letter
// and digit keys are plain lowercase ASCII (e.g. 'w', '1'); the named
// constants below cover the special keys DOOM's key config expects.
type KeyCode byte

const (
	KeyRightArrow  KeyCode = 0xae
	KeyLeftArrow   KeyCode = 0xac
	KeyUpArrow     KeyCode = 0xad
	KeyDownArrow   KeyCode = 0xaf
	KeyStrafeLeft  KeyCode = 0xa0
	KeyStrafeRight KeyCode = 0xa1
	KeyUse         KeyCode = 0xa2
	KeyFire        KeyCode = 0xa3
	KeyEscape      KeyCode = 27
	KeyEnter       KeyCode = 13
	KeyTab         KeyCode = 9
	KeyBackspace   KeyCode = 0x7f
	KeyPause       KeyCode = 0xff
	KeyEquals      KeyCode = 0x3d
	KeyMinus       KeyCode = 0x2d
	KeyRShift      KeyCode = 0x80 + 0x36
	KeyRCtrl       KeyCode = 0x80 + 0x1d
	KeyRAlt        KeyCode = 0x80 + 0x38
	KeyLAlt        KeyCode = KeyRAlt

	KeyF1  KeyCode = 0x80 + 0x3b
	KeyF2  KeyCode = 0x80 + 0x3c
	KeyF3  KeyCode = 0x80 + 0x3d
	KeyF4  KeyCode = 0x80 + 0x3e
	KeyF5  KeyCode = 0x80 + 0x3f
	KeyF6  KeyCode = 0x80 + 0x40
	KeyF7  KeyCode = 0x80 + 0x41
	KeyF8  KeyCode = 0x80 + 0x42
	KeyF9  KeyCode = 0x80 + 0x43
	KeyF10 KeyCode = 0x80 + 0x44
	KeyF11 KeyCode = 0x80 + 0x57
	KeyF12 KeyCode = 0x80 + 0x58
)
