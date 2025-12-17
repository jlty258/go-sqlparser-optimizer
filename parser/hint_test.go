package parser

import (
	"strings"
	"testing"
)

// TestHintParsing 测试 Hint 解析
func TestHintParsing(t *testing.T) {
	testCases := []struct {
		name        string
		sql         string
		expectHints int
		hintNames   []string
	}{
		{
			name:        "单个Hint无参数",
			sql:         "select /*+ TEE */ * from plat1.atest",
			expectHints: 1,
			hintNames:   []string{"TEE"},
		},
		{
			name:        "单个Hint带参数",
			sql:         "select /*+ FUNC(TEE) */ * from plat1.atest",
			expectHints: 1,
			hintNames:   []string{"FUNC"},
		},
		{
			name:        "多个Hint",
			sql:         "select /*+ JOIN(TEE), FUNC(TEE) */ * from plat1.atest, plat2.btest",
			expectHints: 2,
			hintNames:   []string{"JOIN", "FUNC"},
		},
		{
			name:        "TEE功能-两方乘法",
			sql:         "select /*+ FUNC(TEE) */ MUL(plat1.atest.k,plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
			expectHints: 1,
			hintNames:   []string{"FUNC"},
		},
		{
			name:        "TEE功能-两方乘法求和",
			sql:         "select /*+ FUNC(TEE) */ MULSUM(plat1.atest.k,plat2.btest.k) from plat1.atest, plat2.btest",
			expectHints: 1,
			hintNames:   []string{"FUNC"},
		},
		{
			name:        "FL功能-联邦学习",
			sql:         "SELECT /*+ JOIN(FL) */ SEQUENCE(TRAIN(model_name=HOLR)) FROM plat1.atest, plat2.btest",
			expectHints: 1,
			hintNames:   []string{"JOIN"},
		},
		{
			name:        "LOCAL Hint",
			sql:         "SELECT /*+ LOCAL(FL) */ SEQUENCE(TRAIN(model_name=HOLR)) FROM plat1.atest, plat2.btest",
			expectHints: 1,
			hintNames:   []string{"LOCAL"},
		},
		{
			name:        "LLM Hint",
			sql:         "select /*+ LLM(TEE) */ TRAIN(model_name='llama2_70B') from plat1.atest",
			expectHints: 1,
			hintNames:   []string{"LLM"},
		},
		{
			name:        "HE Hint",
			sql:         "select /*+ JOIN(HE) */ plat1.atest.id from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
			expectHints: 1,
			hintNames:   []string{"JOIN"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 解析 SQL
			result, err := ParseSQLWithAntlr(tc.sql)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			if !result.Success {
				t.Fatalf("解析不成功: %s", result.ErrorMessage)
			}

			// 检查是否为 SELECT 语句
			sqlSelect, ok := result.SqlNode.(*SqlSelect)
			if !ok {
				t.Fatalf("不是 SELECT 语句: %T", result.SqlNode)
			}

			// 检查 Hint 数量
			if len(sqlSelect.Hints) != tc.expectHints {
				t.Errorf("期望 %d 个 Hint，实际得到 %d 个", tc.expectHints, len(sqlSelect.Hints))
				for i, hint := range sqlSelect.Hints {
					t.Logf("  Hint %d: %s", i, hint.ToString())
				}
			}

			// 检查 Hint 名称
			if len(sqlSelect.Hints) == len(tc.hintNames) {
				for i, expectedName := range tc.hintNames {
					if sqlSelect.Hints[i].Name != expectedName {
						t.Errorf("Hint %d: 期望名称 %s，实际得到 %s", i, expectedName, sqlSelect.Hints[i].Name)
					}
				}
			}

			// 打印 ToString 结果
			t.Logf("✅ 解析成功")
			t.Logf("   SQL: %s", tc.sql[:min(80, len(tc.sql))])
			t.Logf("   Hint数量: %d", len(sqlSelect.Hints))
			for i, hint := range sqlSelect.Hints {
				t.Logf("   Hint %d: %s (参数: %d 个)", i+1, hint.Name, len(hint.Parameters))
				if len(hint.Parameters) > 0 {
					for j, param := range hint.Parameters {
						t.Logf("     参数 %d: %s", j+1, param.ToString())
					}
				}
			}
			t.Logf("   ToString: %s", sqlSelect.ToString())
		})
	}
}

// TestHintInMPCSQL 测试 MPC SQL 中的 Hint
func TestHintInMPCSQL(t *testing.T) {
	// 从之前跳过的测试中提取
	skipTests := []struct {
		name     string
		sql      string
		category string
	}{
		{
			name:     "TEE两方乘法",
			category: "TEE功能",
			sql:      "select /*+ FUNC(TEE) */ MUL(plat1.atest.k,plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},
		{
			name:     "TEE两方乘法求和",
			category: "TEE功能",
			sql:      "select /*+ FUNC(TEE) */ MULSUM(plat1.atest.k,plat2.btest.k) from plat1.atest, plat2.btest",
		},
	}

	passedCount := 0
	for _, tc := range skipTests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseSQLWithAntlr(tc.sql)
			if err != nil {
				t.Errorf("解析失败: %v", err)
				return
			}

			if !result.Success {
				t.Errorf("解析不成功: %s", result.ErrorMessage)
				return
			}

			sqlSelect, ok := result.SqlNode.(*SqlSelect)
			if !ok {
				t.Errorf("不是 SELECT 语句")
				return
			}

			if len(sqlSelect.Hints) == 0 {
				t.Errorf("未解析到 Hint")
				return
			}

			passedCount++
			t.Logf("✅ 成功解析包含 Hint 的 MPC SQL")
			t.Logf("   类别: %s", tc.category)
			t.Logf("   Hint: %s", sqlSelect.Hints[0].ToString())
		})
	}

	t.Logf("\n" + strings.Repeat("=", 80))
	t.Logf("Hint 解析测试总结")
	t.Logf(strings.Repeat("=", 80))
	t.Logf("通过: %d/%d", passedCount, len(skipTests))
	t.Logf(strings.Repeat("=", 80))
}

