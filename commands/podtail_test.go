package commands

import testing

func Test_ReadManifest(t *testing.T) {
	path := "testdata/azure_simple_vm.pp"
	_, err := New(path)
	if err != nil {
		t.Fail()
	}
}