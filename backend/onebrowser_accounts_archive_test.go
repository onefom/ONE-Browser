package backend

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAccountRARAndPackageRoundTrip(t *testing.T) {
	items := []OneBrowserAccountRecord{{"account-id", "测试账号", "测试密码!123", "备注"}}
	payload, _ := json.Marshal(oneBrowserAccountPackage{"OneBrowserAccounts", 1, items})
	rar := encodeAccountRAR(payload)
	if !bytes.HasPrefix(rar, accountRARSignature) {
		t.Fatal("not RAR5")
	}
	got, err := decodeAccountPackage(rar)
	if err != nil || !reflect.DeepEqual(got, items) {
		t.Fatal(got, err)
	}
	if dir := os.Getenv("ONEBROWSER_RAR_FIXTURE_DIR"); dir != "" {
		if err = os.WriteFile(filepath.Join(dir, "account-fixture.rar"), rar, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, cut := range []int{0, 7, 15, len(rar) - 1} {
		if _, err = decodeAccountPackage(rar[:cut]); err == nil {
			t.Fatal("truncated archive accepted", cut)
		}
	}
	for _, pos := range []int{12, 30, len(rar) - 13} {
		bad := append([]byte{}, rar...)
		bad[pos] ^= 1
		if _, err = decodeAccountPackage(bad); err == nil {
			t.Fatal("corruption accepted", pos)
		}
	}
	if _, err = decodeAccountPackage(append(rar, 'x')); err == nil {
		t.Fatal("trailing content accepted")
	}
	if _, err = decodeAccountPackage([]byte(`{"format":"Other","version":1,"accounts":[]}`)); err == nil {
		t.Fatal("foreign schema accepted")
	}
	if _, err = decodeAccountPackage([]byte(`{"format":"OneBrowserAccounts","version":1,"accounts":[{"service":"","identity":"x"}]}`)); err == nil {
		t.Fatal("invalid name accepted")
	}
}
