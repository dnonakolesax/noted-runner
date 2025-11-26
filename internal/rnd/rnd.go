package rnd

import (
	unsafeRand "math/rand/v2"
)

//nolint:gochecknoglobals // нельзя сделать массив константой
var byteChoice = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// NotSafeGenRandomString НЕ ИСПОЛЬЗОВАТЬ ТАМ, ГДЕ НУЖЕН КРИПТОСТОЙКИЙ РАНДОМ (НАПРИМЕР, ДЛЯ STATE И PKCE).
func NotSafeGenRandomString(length uint) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = byteChoice[unsafeRand.IntN(len(byteChoice))] //nolint:gosec // не тратимся на сисколы там, где не надо
	}
	return b
}
