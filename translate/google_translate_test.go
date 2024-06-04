package translate

import (
	"github.com/go-ini/ini"
	"testing"
)

func TestCreateI18nFile(t *testing.T) {
	err := CreateI18nFile(ini.Empty(), "")

	if err != nil {
		t.Fatal(err.Error())
	}
}
