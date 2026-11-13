package conversation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexProtocolKeepsToolsInCore(t *testing.T) {
	home := t.TempDir()
	server := filepath.Join(home, "server.py")
	script := `import sys,json
for line in sys.stdin:
 r=json.loads(line); method=r.get('method'); result={}
 if method=='account/read': result={'account':{'type':'chatgpt'}}
 if method=='thread/start': result={'thread':{'id':'thread1'}}
 if method=='turn/start': result={'turn':{'id':'turn1'}}
 if 'id' in r: print(json.dumps({'id':r['id'],'result':result}),flush=True)
 if method=='turn/start':
  content='Olá!'
  if 'CASE_TOOL' in str(r): content='{"jarvis_tool":"calculator","input":"25 * 4"}'
  if 'CASE_NATIVE' in str(r):
   print(json.dumps({'method':'item/started','params':{'item':{'type':'commandExecution'}}}),flush=True)
  print(json.dumps({'method':'item/agentMessage/delta','params':{'delta':content}}),flush=True)
  print(json.dumps({'method':'turn/completed','params':{'turn':{'status':'completed'}}}),flush=True)
`
	if e := os.WriteFile(server, []byte(script), 0600); e != nil {
		t.Fatal(e)
	}
	bin := filepath.Join(home, "fake-codex")
	if e := os.WriteFile(bin, []byte("#!/bin/sh\nexec python3 '"+server+"'\n"), 0700); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		input  string
		calls  int
		failed bool
	}{{"normal", 0, false}, {"CASE_TOOL", 1, false}, {"CASE_NATIVE", 0, true}} {
		t.Run(tc.input, func(t *testing.T) {
			var output strings.Builder
			calls, e := (Codex{Home: home, Binary: bin}).Stream(context.Background(), []Message{{Role: "user", Content: tc.input}}, func(s string) error { output.WriteString(s); return nil })
			if (e != nil) != tc.failed {
				t.Fatalf("unexpected error: %v", e)
			}
			if len(calls) != tc.calls {
				t.Fatalf("calls=%v", calls)
			}
			if tc.calls == 1 && (calls[0].Function.Name != "calculator" || output.Len() != 0) {
				t.Fatalf("tool envelope was spoken or lost: %v %s", calls, output.String())
			}
			if tc.input == "normal" && output.String() != "Olá!" {
				t.Fatal(output.String())
			}
		})
	}
}

func TestCodexQuotaIsExplicit(t *testing.T) {
	e := codexFailure([]byte(`{"error":{"codexErrorInfo":"usageLimitExceeded","message":"Try later"}}`))
	if e.Error() != "quota_exhausted" {
		t.Fatal(e)
	}
}
