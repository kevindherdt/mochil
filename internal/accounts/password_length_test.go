package accounts

import (
	"strings"
	"testing"
)

// bcrypt reads at most 72 bytes, so a longer password has to be refused as invalid instead of failing deeper down.
func TestAPasswordLongerThanBcryptCanReadIsInvalid(t *testing.T) {
	for _, length := range []int{9, 10, 72, 73, 1000} {
		want := length >= 10 && length <= 72
		if got := validNewPassword(strings.Repeat("a", length)); got != want {
			t.Errorf("%d bytes: valid = %v, want %v", length, got, want)
		}
	}
}
