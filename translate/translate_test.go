package translate

import "testing"

func TestCreateI18nFile(t *testing.T) {
	err := CreateI18nFile("")

	if err != nil {
		t.Fatal(err.Error())
	}
}
