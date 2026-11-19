// Package isbn finds the ISBN a book belongs to by reading the book, and says why it chose it. Where a book names
// several (its own and those of other volumes in an "also by" list), each is ranked with the reasons for its rank, so a
// person, or later a smarter helper, can judge a close call.
package isbn

import "strings"

// Clean keeps the digits (and an ISBN-10's X check digit) of something that looks like an ISBN.
func Clean(text string) string {
	var out []byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c >= '0' && c <= '9':
			out = append(out, c)
		case (c == 'X' || c == 'x') && len(out) == 9:
			// X is the check digit of an ISBN-10, so it follows exactly nine digits.
			// Requiring it to be the last byte dropped every "…-X (pbk.)" imprint line.
			out = append(out, 'X')
		}
	}
	return string(out)
}

func valid13(digits string) bool {
	if len(digits) != 13 || !(strings.HasPrefix(digits, "978") || strings.HasPrefix(digits, "979")) {
		return false
	}
	sum := 0
	for i := 0; i < 13; i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += int(digits[i]-'0') * weight
	}
	return sum%10 == 0
}

func valid10(digits string) bool {
	if len(digits) != 10 {
		return false
	}
	sum := 0
	for i := 0; i < 10; i++ {
		var value int
		switch c := digits[i]; {
		case c >= '0' && c <= '9':
			value = int(c - '0')
		case c == 'X' && i == 9:
			value = 10
		default:
			return false
		}
		sum += value * (10 - i)
	}
	return sum%11 == 0
}

func check13(first12 string) string {
	sum := 0
	for i := 0; i < 12; i++ {
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += int(first12[i]-'0') * weight
	}
	return string(rune('0' + (10-sum%10)%10))
}

// Normalize returns the ISBN-13 for anything that is a correct ISBN-10 or ISBN-13 (hyphens and spaces allowed).
func Normalize(text string) (string, bool) {
	digits := Clean(text)
	switch {
	case valid13(digits):
		return digits, true
	case valid10(digits):
		base := "978" + digits[:9]
		return base + check13(base), true
	}
	return "", false
}
