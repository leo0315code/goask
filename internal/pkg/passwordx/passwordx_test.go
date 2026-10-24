package passwordx

import "testing"

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("secret123", 4) // 低 cost 加速测试
	if err != nil {
		t.Fatalf("Hash 失败: %v", err)
	}
	if hash == "secret123" || len(hash) == 0 {
		t.Fatal("哈希不应等于明文且非空")
	}
	if !Verify(hash, "secret123") {
		t.Fatal("正确密码应校验通过")
	}
	if Verify(hash, "wrong-pass") {
		t.Fatal("错误密码不应校验通过")
	}
}

func TestHashSamePasswordDifferentHash(t *testing.T) {
	h1, _ := Hash("same-password", 4)
	h2, _ := Hash("same-password", 4)
	if h1 == h2 {
		t.Fatal("同密码两次哈希应不同（自带盐）")
	}
}
