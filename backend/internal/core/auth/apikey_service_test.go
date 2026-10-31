// 本文件验证 API 密钥的摘要、随机生成、文本拆分和角色限制。
// 测试只调用纯函数，不连接数据库，也不替代完整请求的身份认证测试。
package auth

import (
	"testing"
)

// 相同输入须得到相同摘要，所列不同输入及占位密钥的摘要须互不相同。
// 这里验证具体样本，不把有限样本作为所有输入绝不碰撞的证明。
func TestHashSecret_DeterministicAndDistinct(t *testing.T) {
	h1 := hashSecret("secret-a")
	h2 := hashSecret("secret-a")
	h3 := hashSecret("secret-b")
	if h1 != h2 {
		t.Fatalf("expected hashSecret to be deterministic, got %q vs %q", h1, h2)
	}
	if h1 == h3 {
		t.Fatalf("expected different secrets to hash differently")
	}
	if h1 == dummySecretHash {
		t.Fatalf("real secret hash must not collide with the dummy hash used for timing equalization")
	}
}

// 连续生成两组前缀和密钥，检查无错误、非空且两次值不同。
// 测试未通过统计样本评价随机分布，也不验证数据库前缀冲突重试。
func TestGenerateKeyToken_ProducesNonEmptyDistinctValues(t *testing.T) {
	prefix1, secret1, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	prefix2, secret2, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	if prefix1 == "" || secret1 == "" {
		t.Fatalf("expected non-empty prefix/secret")
	}
	if prefix1 == prefix2 || secret1 == secret2 {
		t.Fatalf("expected distinct values across calls")
	}
}

// 按生产令牌格式拼接随机值，再拆分还原，确认开头和分隔位置相互配合。
func TestParseKeyToken_RoundTrip(t *testing.T) {
	prefix, secret, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	token := apiKeyTokenPrefix + prefix + "_" + secret

	gotPrefix, gotSecret, ok := parseKeyToken(token)
	if !ok {
		t.Fatalf("expected token to parse")
	}
	if gotPrefix != prefix || gotSecret != secret {
		t.Fatalf("got (%q, %q), want (%q, %q)", gotPrefix, gotSecret, prefix, secret)
	}
}

// 列举缺少开头、分隔符或任意非空组成部分的输入，要求解析统一拒绝。
func TestParseKeyToken_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"not-a-key-token",
		"lw_missingsecret",
		"lw__",
		"lw_prefixonly_",
		"lw__secretonly",
	}
	// 每个样本独立调用解析器，不依赖上一个样本留下的状态。
	for _, tc := range cases {
		if _, _, ok := parseKeyToken(tc); ok {
			t.Fatalf("expected %q to be rejected as malformed", tc)
		}
	}
}

// 穷举三个已知角色的九种组合，确认结果始终选择两者权限较低的角色。
func TestIntersectRole_NeverWidensPermission(t *testing.T) {
	tests := []struct {
		userRole, keyRole, want string
	}{
		{RoleAdmin, RoleAdmin, RoleAdmin},
		{RoleAdmin, RoleEditor, RoleEditor},
		{RoleAdmin, RoleViewer, RoleViewer},
		{RoleEditor, RoleAdmin, RoleEditor},
		{RoleEditor, RoleEditor, RoleEditor},
		{RoleEditor, RoleViewer, RoleViewer},
		{RoleViewer, RoleAdmin, RoleViewer},
		{RoleViewer, RoleEditor, RoleViewer},
		{RoleViewer, RoleViewer, RoleViewer},
	}
	for _, tc := range tests {
		got := intersectRole(tc.userRole, tc.keyRole)
		if got != tc.want {
			t.Errorf("intersectRole(%q, %q) = %q, want %q", tc.userRole, tc.keyRole, got, tc.want)
		}
	}
}

// 分别把账号角色和密钥角色设为未知值，确认两条路径均使用只读角色。
func TestIntersectRole_UnknownRoleFailsSafeToViewer(t *testing.T) {
	if got := intersectRole("bogus", RoleAdmin); got != RoleViewer {
		t.Errorf("intersectRole(bogus, admin) = %q, want %q (fail safe)", got, RoleViewer)
	}
	if got := intersectRole(RoleAdmin, "bogus"); got != RoleViewer {
		t.Errorf("intersectRole(admin, bogus) = %q, want %q (fail safe)", got, RoleViewer)
	}
}

// generateKeyToken 为测试依次生成前缀和密钥，任一步失败都返回空组成部分和错误。
// 它不执行存储写入，测试自行按生产格式拼接完整令牌。
func generateKeyToken() (prefix string, secret string, err error) {
	prefix, err = generatePrefix()
	if err != nil {
		return "", "", err
	}
	secret, err = generateSecret()
	if err != nil {
		return "", "", err
	}
	return prefix, secret, nil
}
