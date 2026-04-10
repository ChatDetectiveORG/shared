package utils

import "unicode/utf16"

func TgLen(s string) int {
    return len(utf16.Encode([]rune(s)))
}
