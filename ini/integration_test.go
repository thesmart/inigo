package ini

import (
	"strings"
	"testing"
)

func TestUnmarshalIniFileIntermediate_CircularConf(t *testing.T) {
	_, err := unmarshalIniFileIntermediate("testdata/circular.conf")
	if err == nil {
		t.Fatal("expected circular include error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected error to mention 'circular', got: %v", err)
	}
}
