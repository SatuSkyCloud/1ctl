package commands

import "testing"

func TestOpenBrowser_RejectsEmptyURL(t *testing.T) {
	if err := openBrowser(""); err == nil {
		t.Error("openBrowser(\"\") should return an error")
	}
}
