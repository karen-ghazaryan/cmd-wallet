package encrypting

import "testing"

func TestNew(t *testing.T) {
	tests := []string{
		"throw couple humor orbit snap design trip august toss quick bronze fiscal",
		"AES256Key-32Characters1234567890",
		"tprv8iqD2cFmt4GVydPNTWR5itazMCgatP3UBoCVLxUpHBhXwAyMMVgmGgYBWE2DGVPR5GhEfPXXXSBQCt8MaUbTKQdBUBmFg5jkLhCV6j2FPEs",
	}

	for _, msg := range tests {
		testMsg := []byte(msg)
		t.Log("testMsg", testMsg)
		enc, err := New("secret")
		if err != nil {
			t.Fatal(err)
		}

		encMsg, err := enc.Encrypt(testMsg)
		if err != nil {
			t.Error("Encrypt", err)
		}

		res, err := enc.Decrypt(encMsg)
		if err != nil {
			t.Error(err)
		}

		if string(res) != msg {
			t.Errorf("expected: %s, \ngot: %s", msg, res)
		}
	}
}