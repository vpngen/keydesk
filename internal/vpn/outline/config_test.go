package outline

import "testing"

func TestAccessKey(t *testing.T) {
	config, err := generator{}.Generate()
	if err != nil {
		t.Fatal(err)
	}
	accessKey := config.UserConfig("test", "example.com", 443)
	t.Log(accessKey)
}
