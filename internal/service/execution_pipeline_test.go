package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
)

func TestBuildExecutionRunPlan(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		plan := buildExecutionRunPlan(false, false)
		if !plan.RunLocal || plan.RunRemote || plan.MergeResults {
			t.Fatalf("unexpected local plan: %+v", plan)
		}
	})

	t.Run("distributed_remote_only", func(t *testing.T) {
		plan := buildExecutionRunPlan(true, false)
		if plan.RunLocal || !plan.RunRemote || plan.MergeResults {
			t.Fatalf("unexpected remote plan: %+v", plan)
		}
	})

	t.Run("distributed_with_master", func(t *testing.T) {
		plan := buildExecutionRunPlan(true, true)
		if !plan.RunLocal || !plan.RunRemote || !plan.MergeResults {
			t.Fatalf("unexpected distributed+master plan: %+v", plan)
		}
	})
}

func TestCompareExecutionEnvironments(t *testing.T) {
	base := executionEnvironmentReport{
		Node:                  "master-local",
		AgentVersion:          "1.0.0",
		JMeterVersion:         "5.6.3",
		PluginFingerprint:     "plugin-a",
		PropertiesFingerprint: "props-a",
	}
	others := []executionEnvironmentReport{
		{
			Node:                  "slave-1",
			AgentVersion:          "1.0.1",
			JMeterVersion:         "5.6.4",
			PluginFingerprint:     "plugin-b",
			PropertiesFingerprint: "props-b",
		},
	}

	mismatches := compareExecutionEnvironments(base, others)
	if len(mismatches) != 4 {
		t.Fatalf("expected 4 mismatches, got %d: %v", len(mismatches), mismatches)
	}
}

func TestNormalizePropertiesContent(t *testing.T) {
	contentA := strings.Join([]string{
		"# comment",
		"threads = 100",
		"",
		"host=example.com",
	}, "\n")
	contentB := strings.Join([]string{
		"! another comment",
		"host = example.com",
		"threads=100",
		"",
	}, "\n")

	gotA := normalizePropertiesContent([]byte(contentA))
	gotB := normalizePropertiesContent([]byte(contentB))
	if gotA != gotB {
		t.Fatalf("expected normalized content to match, got %q != %q", gotA, gotB)
	}
}

func TestParseJMeterVersion(t *testing.T) {
	output := "Version 5.6.3\nCopyright ..."
	if got := parseJMeterVersion(output); got != "5.6.3" {
		t.Fatalf("expected 5.6.3, got %q", got)
	}
}

func TestAbsolutizeRuntimeDependencyPaths(t *testing.T) {
	baseDir := "/tmp/demo"
	content := `<CSVDataSet><stringProp name="filename">Test_tokens3.csv</stringProp></CSVDataSet>`
	got := absolutizeRuntimeDependencyPaths(content, baseDir)
	if !strings.Contains(got, "/tmp/demo/Test_tokens3.csv") {
		t.Fatalf("expected absolute dependency path, got %q", got)
	}
}

func TestReplaceFileDependencyPaths(t *testing.T) {
	content := `<stringProp name="filename">Test_tokens3.csv</stringProp>`
	got := replaceFileDependencyPaths(content, "/opt/jmeter/csv-data", []string{"Test_tokens3.csv"})
	if !strings.Contains(got, "/opt/jmeter/csv-data/Test_tokens3.csv") {
		t.Fatalf("expected remote dependency path, got %q", got)
	}
}

func TestReplaceFileDependencyPathsWithMapUsesExactMatch(t *testing.T) {
	content := strings.Join([]string{
		`<stringProp name="filename">data/a/users.csv</stringProp>`,
		`<stringProp name="filename">data/b/users.csv</stringProp>`,
	}, "")

	got := replaceFileDependencyPathsWithMap(content, "/opt/jmeter/csv-data", map[string]string{
		"data/a/users.csv": "exec10_dep_1_aaaa_users.csv",
		"data/b/users.csv": "exec10_dep_2_bbbb_users.csv",
	})

	if !strings.Contains(got, "/opt/jmeter/csv-data/exec10_dep_1_aaaa_users.csv") {
		t.Fatalf("expected first mapped dependency path, got %q", got)
	}
	if !strings.Contains(got, "/opt/jmeter/csv-data/exec10_dep_2_bbbb_users.csv") {
		t.Fatalf("expected second mapped dependency path, got %q", got)
	}
}

func TestExtractCSVDataSetReferences(t *testing.T) {
	jmx := strings.Join([]string{
		`<CSVDataSet testclass="CSVDataSet">`,
		`<stringProp name="filename">data/users.csv</stringProp>`,
		`<boolProp name="ignoreFirstLine">true</boolProp>`,
		`<stringProp name="shareMode">shareMode.group</stringProp>`,
		`</CSVDataSet>`,
		`<CSVDataSet testclass="CSVDataSet">`,
		`<stringProp name="filename">dict/areas.csv</stringProp>`,
		`<boolProp name="ignoreFirstLine">false</boolProp>`,
		`</CSVDataSet>`,
	}, "")

	refs := extractCSVDataSetReferences(jmx)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].Filename != "data/users.csv" || !refs[0].IgnoreFirst {
		t.Fatalf("unexpected first ref: %+v", refs[0])
	}
	if refs[1].Filename != "dict/areas.csv" || refs[1].IgnoreFirst {
		t.Fatalf("unexpected second ref: %+v", refs[1])
	}
}

func TestReplaceCSVDataSetPathsWithMap(t *testing.T) {
	jmx := strings.Join([]string{
		`<CSVDataSet testclass="CSVDataSet">`,
		`<stringProp name="filename">data/a/users.csv</stringProp>`,
		`<stringProp name="shareMode">shareMode.group</stringProp>`,
		`</CSVDataSet>`,
		`<CSVDataSet testclass="CSVDataSet">`,
		`<stringProp name="filename">data/b/users.csv</stringProp>`,
		`<stringProp name="shareMode">shareMode.current_thread</stringProp>`,
		`</CSVDataSet>`,
	}, "")

	got := replaceCSVDataSetPathsWithMap(jmx, "/opt/jmeter/csv-data", map[string]string{
		"data/a/users.csv": "exec9_csv_1_aaaa_users.csv",
		"data/b/users.csv": "exec9_csv_2_bbbb_users.csv",
	})

	if !strings.Contains(got, "/opt/jmeter/csv-data/exec9_csv_1_aaaa_users.csv") {
		t.Fatalf("expected first CSV path replaced, got %q", got)
	}
	if !strings.Contains(got, "/opt/jmeter/csv-data/exec9_csv_2_bbbb_users.csv") {
		t.Fatalf("expected second CSV path replaced, got %q", got)
	}
	if strings.Count(got, "shareMode.all") != 2 {
		t.Fatalf("expected shareMode updated for both split CSV datasets, got %q", got)
	}
}

func TestSplitCSVWithoutHeader(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.csv")
	content := "a1,b1\na2,b2\na3,b3\na4,b4\n"
	if err := os.WriteFile(source, []byte(content), 0644); err != nil {
		t.Fatalf("write source csv: %v", err)
	}

	parts, err := SplitCSV(source, 2, false, dir, "runtime.csv")
	if err != nil {
		t.Fatalf("SplitCSV failed: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	firstPart, err := os.ReadFile(parts[0])
	if err != nil {
		t.Fatalf("read first part: %v", err)
	}
	if strings.HasPrefix(string(firstPart), "a1,b1\na1,b1") {
		t.Fatalf("unexpected duplicated first row in split result: %q", string(firstPart))
	}
	if strings.Count(strings.TrimSpace(string(firstPart)), "\n") != 1 {
		t.Fatalf("expected exactly 2 data rows in first split, got %q", string(firstPart))
	}
}

func TestMergeJTLFiles(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.jtl")
	fileB := filepath.Join(dir, "b.jtl")
	out := filepath.Join(dir, "merged.jtl")

	content := "timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,Encoding,IdleTime,Connect\n"
	if err := os.WriteFile(fileA, []byte(content+"1,10,apiA,200,OK,t1,text,true,,10,1,1,1,https://a,2,utf-8,0,1\n"), 0644); err != nil {
		t.Fatalf("write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(content+"2,12,apiB,500,ERR,t2,text,false,boom,12,2,1,1,https://b,3,utf-8,0,1\n"), 0644); err != nil {
		t.Fatalf("write fileB: %v", err)
	}

	if err := mergeJTLFiles([]string{fileA, fileB}, out); err != nil {
		t.Fatalf("mergeJTLFiles failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read merged file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), string(data))
	}
}

func TestParseJTLResultsPrefersTransactionThroughput(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "result.jtl")
	content := strings.Join([]string{
		"timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,Encoding,IdleTime,Connect",
		"1000,100,HTTP Request,200,OK,t1,text,true,,10,1,1,1,https://example.com/api,5,utf-8,0,1",
		"1000,100,事务控制器,200,OK,t1,text,true,,10,1,1,1,,5,utf-8,0,1",
		"2000,100,HTTP Request,200,OK,t1,text,true,,10,1,1,1,https://example.com/api,5,utf-8,0,1",
		"2000,100,事务控制器,200,OK,t1,text,true,,10,1,1,1,,5,utf-8,0,1",
	}, "\n") + "\n"
	if err := os.WriteFile(resultPath, []byte(content), 0644); err != nil {
		t.Fatalf("write result file: %v", err)
	}

	summaryJSON, err := parseJTLResults(resultPath)
	if err != nil {
		t.Fatalf("parseJTLResults failed: %v", err)
	}

	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(summaryJSON), &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}

	if got := summary["primary_throughput_field"]; got != "transaction_tps" {
		t.Fatalf("expected transaction_tps, got %v", got)
	}
	if got := int(summary["transaction_samples"].(float64)); got != 2 {
		t.Fatalf("expected 2 transaction samples, got %d", got)
	}
	if got := int(summary["request_samples"].(float64)); got != 2 {
		t.Fatalf("expected 2 request samples, got %d", got)
	}
}

func TestGetExecutionErrorsCreatesAndUsesIndex(t *testing.T) {
	dir := t.TempDir()
	config.GlobalConfig.Dirs.Data = filepath.Join(dir, "data")
	config.GlobalConfig.Dirs.Results = filepath.Join(dir, "results")
	if err := os.MkdirAll(config.GlobalConfig.Dirs.Data, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(config.GlobalConfig.Dirs.Results, 0755); err != nil {
		t.Fatalf("create results dir: %v", err)
	}

	if database.DB != nil {
		_ = database.CloseDB()
	}
	if err := database.InitDB(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer func() {
		_ = database.CloseDB()
	}()

	now := "2026-04-11 10:00:00"
	if _, err := database.DB.Exec(
		"INSERT INTO scripts (id, name, description, file_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		1, "demo", "", filepath.Join(dir, "demo.jmx"), now, now,
	); err != nil {
		t.Fatalf("insert script: %v", err)
	}

	resultDir := filepath.Join(config.GlobalConfig.Dirs.Results, "1")
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		t.Fatalf("create result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, "result.jtl")
	jtlContent := strings.Join([]string{
		"timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,Encoding,IdleTime,Connect",
		"1712800800000,120,prodInfo,200,OK,t1,text,false,assert failed,100,10,1,1,https://example.com/api,10,utf-8,0,2",
	}, "\n") + "\n"
	if err := os.WriteFile(resultPath, []byte(jtlContent), 0644); err != nil {
		t.Fatalf("write result file: %v", err)
	}

	if _, err := database.DB.Exec(
		"INSERT INTO executions (id, script_id, script_name, slave_ids, status, start_time, remarks, result_path, report_path, summary_data, log_path, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		1, 1, "demo", "[]", "success", now, "", resultPath, filepath.Join(resultDir, "report"), "", filepath.Join(resultDir, "execution.log"), now,
	); err != nil {
		t.Fatalf("insert execution: %v", err)
	}

	errorAnalysisCacheMu.Lock()
	errorAnalysisCache = make(map[int64]errorAnalysisCacheEntry)
	errorAnalysisCacheMu.Unlock()

	analysis, err := GetExecutionErrors(1)
	if err != nil {
		t.Fatalf("GetExecutionErrors first call failed: %v", err)
	}
	if analysis.TotalErrors != 1 || analysis.ErrorRate != 100 {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}

	indexPath := errorAnalysisIndexPath(resultPath)
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected index file to exist: %v", err)
	}

	errorAnalysisCacheMu.Lock()
	errorAnalysisCache = make(map[int64]errorAnalysisCacheEntry)
	errorAnalysisCacheMu.Unlock()

	analysis2, err := GetExecutionErrors(1)
	if err != nil {
		t.Fatalf("GetExecutionErrors second call failed: %v", err)
	}
	if analysis2.TotalErrors != 1 || len(analysis2.Records) != 1 {
		t.Fatalf("unexpected indexed analysis: %+v", analysis2)
	}
}
