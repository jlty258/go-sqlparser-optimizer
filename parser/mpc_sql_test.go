package parser

import (
	"fmt"
	"strings"
	"testing"
)

// TestMPCV2PqlSet 测试从 MPCV2PqlSet.java 提取的所有 SQL 语句
// 这些 SQL 来自真实的多方安全计算场景
func TestMPCV2PqlSet(t *testing.T) {
	// 定义测试用例结构
	type testCase struct {
		name     string
		sql      string
		category string
		skip     bool // 是否跳过测试
		reason   string // 跳过的原因
	}

	testCases := []testCase{
		// ============= 单方计算 - 基础查询 =============
		{
			name:     "单表条件查询",
			category: "单方计算",
			sql:      "select plat1.atest.k from plat1.atest where plat1.atest.id = 1",
		},
		{
			name:     "单表聚合函数",
			category: "单方计算",
			sql:      "select count(plat1.atest.k), max(plat1.atest.k), avg(plat1.atest.k) from plat1.atest",
		},
		{
			name:     "单表求和",
			category: "单方计算",
			sql:      "select SUM(plat1.atest.k) from plat1.atest",
		},
		{
			name:     "简单子查询",
			category: "单方计算",
			sql:      "select temp.a1 from (select plat1.atest.a1 from plat1.atest) temp",
		},

		// ============= 多方计算 - 基础查询 =============
		{
			name:     "两方简单子查询",
			category: "多方计算",
			sql:      "select plat2.btest.b1, tmp_table.id from plat1.atest, plat2.btest,(select id, a1 from plat1.atest ) tmp_table where plat1.atest.id= plat2.btest.id and tmp_table.id= plat2.btest.id",
		},
		{
			name:     "两方复杂子查询",
			category: "多方计算",
			sql:      "select plat2.btest.b1, tmp_table.id from plat1.atest, plat2.btest,(select id, cnt, tot_val from (select id, count(a1) as cnt, sum(a1) as tot_val from plat1.atest group by id ) tmp_inner ) tmp_table where plat1.atest.id= plat2.btest.id and tmp_table.id= plat2.btest.id",
		},
		{
			name:     "两方子查询计算",
			category: "多方计算",
			sql:      "select plat1.atest.a1, tmp_table.id * 2 + plat2.btest.b2 from plat1.atest, plat2.btest,( select id, cnt, tot_val from ( select id, count(a1) as cnt, sum(a1) as tot_val from plat1.atest group by id) tmp_inner ) tmp_table where plat1.atest.id= plat2.btest.id and tmp_table.id= plat2.btest.id",
		},

		// ============= 多方计算 - 多表关联 =============
		{
			name:     "三方关联",
			category: "多方关联",
			sql:      "select plat1.atest.k, plat2.btest.b2 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id = plat2.btest.id and plat1.atest.a1 = 1",
		},
		{
			name:     "三方关联（两个关联条件）",
			category: "多方关联",
			sql:      "select plat1.atest.k, plat2.btest.b2 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id = plat2.btest.id and plat2.btest.id = plat3.ctest.id and plat1.atest.a1 = 1",
		},

		// ============= 多方计算 - 数学运算 =============
		{
			name:     "两方乘法运算",
			category: "数学运算",
			sql:      "select plat1.atest.k, plat1.atest.a1, plat2.btest.b1, 2 * plat1.atest.k*plat2.btest.k + 3 * plat1.atest.a1 from plat1.atest, plat2.btest where plat1.atest.id = plat2.btest.id",
		},
		{
			name:     "三方加法相加",
			category: "数学运算",
			sql:      "SELECT plat1.atest.k, plat1.atest.a1, plat2.btest.b1, plat2.btest.id, plat1.atest.a1 + plat3.ctest.c3 FROM plat1.atest, plat2.btest, plat3.ctest WHERE plat1.atest.id = plat2.btest.id AND plat3.ctest.id = plat2.btest.id",
		},
		{
			name:     "三方乘法运算",
			category: "数学运算",
			sql:      "select 2 * plat1.atest.k*plat2.btest.k + 3 * plat1.atest.a1*plat3.ctest.c3 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id=plat2.btest.id AND plat3.ctest.id = plat2.btest.id",
		},
		{
			name:     "三方乘法运算带条件",
			category: "数学运算",
			sql:      "select plat3.ctest.c3, 2 * plat1.atest.k*plat2.btest.k + 3 * plat1.atest.a1*plat3.ctest.c3 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id = plat2.btest.id AND plat3.ctest.id = plat2.btest.id and plat1.atest.a1 = 1",
		},
		{
			name:     "三方乘法运算多条件",
			category: "数学运算",
			sql:      "select plat3.ctest.id, plat3.ctest.c3, 2 * plat1.atest.k*plat2.btest.k + 3 * plat1.atest.a1*plat3.ctest.c3 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id = plat2.btest.id and plat2.btest.id = plat3.ctest.id and plat1.atest.a1 = 1",
		},

		// ============= SET 语句（跳过）=============
		{
			name:     "SET语句-多方PSI",
			category: "SET语句",
			sql:      "set engine.software.psi.multi=true; select plat3.ctest.id, plat3.ctest.c3, 2 * plat1.atest.k*plat2.btest.k + 3 * plat1.atest.a1*plat3.ctest.c3 from plat1.atest, plat2.btest,plat3.ctest where plat1.atest.id = plat2.btest.id and plat1.atest.a1 = plat2.btest.b1 and plat2.btest.id = plat3.ctest.id and plat2.btest.b1 = plat3.ctest.c3 and plat1.atest.a1 = 1",
			skip:     true,
			reason:   "包含 SET 语句，需要特殊处理",
		},

		// ============= 权重表场景 =============
		{
			name:     "权重表-两方加权求和",
			category: "权重表",
			sql:      "set engine.software.weight.tables = plat3.ctest_w, plat2.btest_w;select plat1.atest.id, (0.1 * plat1.atest.a1 * plat2.btest_w.w2) + (0.2 * plat2.btest.b1 * plat3.ctest_w.w3) + (0.1 * plat1.atest.a2) + (0.4 * plat2.btest.b2) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
			skip:     true,
			reason:   "包含 SET 语句",
		},
		{
			name:     "两方加权求和",
			category: "数学运算",
			sql:      "select plat1.atest.id, (0.1 * plat1.atest.a1) + (0.2 * plat2.btest.b1) + (0.1 * plat1.atest.a2) + (0.4 * plat2.btest.b2) from plat1.atest, plat2.btest where plat1.atest.id = plat2.btest.id",
		},
		{
			name:     "两方加权求和-字段关联",
			category: "数学运算",
			sql:      "select plat1.atest.id, (0.1 * plat1.atest.a1) + (0.2 * plat2.btest.b1) + (0.1 * plat1.atest.a2) + (0.4 * plat2.btest.b2) from plat1.atest, plat2.btest where plat1.atest.a1 = plat2.btest.b1",
		},
		{
			name:     "两方加权求和带字段",
			category: "数学运算",
			sql:      "select plat1.atest.id, plat2.btest.b1, (0.1 * plat1.atest.a1) + (0.2 * plat2.btest.b1) + (0.1 * plat1.atest.a2) + (0.4 * plat2.btest.b2) from plat1.atest, plat2.btest where plat1.atest.a1 = plat2.btest.b1",
		},

		// ============= 多方计算 - 聚合函数 =============
		{
			name:     "两方乘法求和",
			category: "聚合函数",
			sql:      "select SUM(plat1.atest.k*plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},
		{
			name:     "两方乘法平均值",
			category: "聚合函数",
			sql:      "select AVG(plat1.atest.k*plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},
		{
			name:     "两方乘法最大值",
			category: "聚合函数",
			sql:      "select MAX(plat1.atest.k*plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},
		{
			name:     "两方乘法最小值",
			category: "聚合函数",
			sql:      "select MIN(plat1.atest.k*plat2.btest.k) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},
		{
			name:     "两方计数",
			category: "聚合函数",
			sql:      "select COUNT(plat1.atest.id) from plat1.atest, plat2.btest where plat1.atest.id=plat2.btest.id",
		},

		// ============= 多方计算 - 复杂子查询 =============
		{
			name:     "两方复杂子查询-多临时表",
			category: "复杂子查询",
			sql:      "select plat1.atest.a1, tmp_table1.id as tmp_id1, tmp_table2.id as tmp_id2 from plat1.atest,( select id, count(b2) as cnt, sum(b2) as tot_val from plat2.btest group by id) tmp_table1, ( select id, count(a1) as cnt, sum(a1) as tot_val from plat1.atest group by id ) tmp_table2 where plat1.atest.id= tmp_table1.id and tmp_table1.id= tmp_table2.id",
		},
		{
			name:     "两方子查询计算",
			category: "复杂子查询",
			sql:      "select plat1.atest.k + tmp_table1.id*2 + tmp_table2.id*8 from plat1.atest,( select id, count(b2) as cnt, sum(b2) as tot_val from plat2.btest group by id) tmp_table1, ( select id, count(a1) as cnt, sum(a1) as tot_val from plat1.atest group by id ) tmp_table2 where plat1.atest.id= tmp_table1.id and tmp_table1.id= tmp_table2.id",
		},

		// ============= TEE 功能（带 HINT）=============
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

		// ============= JOIN 语法 =============
		{
			name:     "LEFT OUTER JOIN",
			category: "JOIN",
			sql:      "select plat1.atest.id from plat2.btest left outer join plat1.atest on plat1.atest.id = plat2.btest.id where plat1.atest.id is null",
		},
		{
			name:     "RIGHT OUTER JOIN",
			category: "JOIN",
			sql:      "select plat1.atest.id from plat1.atest right outer join plat2.btest on plat1.atest.id = plat2.btest.id where plat1.atest.id is null",
		},
		{
			name:     "FULL OUTER JOIN",
			category: "JOIN",
			sql:      "select plat1.atest.id from plat1.atest full outer join plat2.btest on plat1.atest.id = plat2.btest.id where plat2.btest.id is null",
		},
		{
			name:     "LEFT OUTER JOIN with NOT NULL",
			category: "JOIN",
			sql:      "select plat1.atest.id as id0 from plat1.atest left outer join plat2.btest on plat1.atest.id = plat2.btest.id where plat2.btest.id is not null",
		},

		// ============= 其他复杂查询 =============
		{
			name:     "自字段相加",
			category: "其他",
			sql:      "select plat1.atest.k + plat1.atest.k from plat1.atest, plat2.btest where plat1.atest.id = plat2.btest.id",
		},
		{
			name:     "自字段相加带别名",
			category: "其他",
			sql:      "select plat1.atest.k + plat1.atest.k as a from plat1.atest, plat2.btest where plat1.atest.id = plat2.btest.id",
		},
		{
			name:     "不等于条件",
			category: "其他",
			sql:      "select plat1.atest.id from plat1.atest, plat2.btest where plat1.atest.id <> plat2.btest.id",
		},
		{
			name:     "三方PSI",
			category: "多方关联",
			sql:      "SELECT plat1.atest.id + plat3.ctest.id FROM plat1.atest, plat2.btest, plat3.ctest WHERE plat1.atest.id = plat2.btest.id AND plat3.ctest.id = plat2.btest.id",
		},
		{
			name:     "简单子查询字段裁剪",
			category: "子查询",
			sql:      "select plat1.atest.a1, tmp_table.id from plat1.atest, plat2.btest,(select id, a1 from plat1.atest ) tmp_table where plat1.atest.id= plat2.btest.id and tmp_table.id= plat2.btest.id",
		},

		// ============= 当前活跃的测试 SQL =============
		{
			name:     "复杂聚合和GROUP BY",
			category: "复杂查询",
			sql:      "select plat1.atest.a1, sum(tmp_table2.tot_val2 + 2 * plat1.atest.a1) as result from plat1.atest,( select id as id2, count(b2) as cnt2, sum(b2) as tot_val2 from plat2.btest group by id) tmp_table2, (select id as id1, count(a1) as cnt1, sum(a1) as tot_val1 from plat1.atest group by id ) tmp_table1 where plat1.atest.id= tmp_table1.id1 and tmp_table1.id1= tmp_table2.id2 group by plat1.atest.a1, plat1.atest.k, tmp_table2.id2",
		},
	}

	// 统计信息
	totalTests := len(testCases)
	passedTests := 0
	failedTests := 0
	skippedTests := 0

	// 按分类统计
	categoryStats := make(map[string]int)
	categoryPassed := make(map[string]int)

	// 运行所有测试
	for i, tc := range testCases {
		testName := fmt.Sprintf("%d_%s_%s", i+1, tc.category, tc.name)
		
		categoryStats[tc.category]++
		
		t.Run(testName, func(t *testing.T) {
			if tc.skip {
				t.Skipf("跳过: %s", tc.reason)
				skippedTests++
				return
			}

			// 解析 SQL
			result, err := ParseSQLWithAntlr(tc.sql)
			
			if err != nil {
				failedTests++
				t.Errorf("解析失败: %v\nSQL: %s", err, tc.sql)
				return
			}

			if !result.Success {
				failedTests++
				t.Errorf("解析不成功: %s\nSQL: %s", result.ErrorMessage, tc.sql)
				return
			}

			if result.SqlNode == nil {
				failedTests++
				t.Errorf("SqlNode 为空\nSQL: %s", tc.sql)
				return
			}

			// 检查是否为 SELECT 语句
			if sqlSelect, ok := result.SqlNode.(*SqlSelect); ok {
				passedTests++
				categoryPassed[tc.category]++
				hintInfo := ""
				if len(sqlSelect.Hints) > 0 {
					hintNames := make([]string, len(sqlSelect.Hints))
					for i, hint := range sqlSelect.Hints {
						hintNames[i] = hint.Name
					}
					hintInfo = fmt.Sprintf("\n   Hints: %v", hintNames)
				}
				t.Logf("✅ 解析成功\n   类别: %s\n   SQL: %s\n   SELECT项数: %d\n   有FROM: %v\n   有WHERE: %v\n   有GROUP BY: %v%s",
					tc.category,
					tc.sql[:min(80, len(tc.sql))],
					len(sqlSelect.SelectList),
					sqlSelect.From != nil,
					sqlSelect.Where != nil,
					len(sqlSelect.GroupBy) > 0,
					hintInfo,
				)
			} else {
				passedTests++
				categoryPassed[tc.category]++
				t.Logf("✅ 解析成功（非SELECT语句）\n   类别: %s\n   类型: %T",
					tc.category,
					result.SqlNode,
				)
			}
		})
	}

	// 输出总体统计
	t.Logf("\n" + strings.Repeat("=", 80))
	t.Logf("MPC SQL 测试总结")
	t.Logf(strings.Repeat("=", 80))
	t.Logf("总测试数: %d", totalTests)
	t.Logf("通过: %d (%.1f%%)", passedTests, float64(passedTests)/float64(totalTests-skippedTests)*100)
	t.Logf("失败: %d (%.1f%%)", failedTests, float64(failedTests)/float64(totalTests-skippedTests)*100)
	t.Logf("跳过: %d", skippedTests)
	
	t.Logf("\n按分类统计:")
	t.Logf(strings.Repeat("-", 80))
	for category, total := range categoryStats {
		passed := categoryPassed[category]
		passRate := 0.0
		if total > 0 {
			passRate = float64(passed) / float64(total) * 100
		}
		t.Logf("%-15s: %2d/%2d 通过 (%.1f%%)", category, passed, total, passRate)
	}
	t.Logf(strings.Repeat("=", 80))
}

// min 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestMPCV2PqlSet_Individual 可以单独测试某个 SQL
func TestMPCV2PqlSet_Individual(t *testing.T) {
	// 这里可以单独测试某个特定的 SQL
	sql := "select plat1.atest.k from plat1.atest where plat1.atest.id = 1"
	
	result, err := ParseSQLWithAntlr(sql)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("解析不成功: %s", result.ErrorMessage)
	}
	
	t.Logf("解析成功: %s", result.SqlNode.ToString())
}







