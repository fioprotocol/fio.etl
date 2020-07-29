package transform

import (
	"encoding/json"
	"strconv"
)

type TraceResult struct {
	Id         string      `json:"id"`
	RecordType string      `json:"record_type"`
	BlockNum   interface{} `json:"block_num"`
	BlockTime  string      `json:"block_timestamp"`
	Trace      FullTrace   `json:"trace"`
}

type FullTrace struct {
	NetUsageWords   string                 `json:"net_usage_words"`
	Scheduled       string                 `json:"scheduled"`
	Partial         map[string]interface{} `json:"partial"`
	AccountRamDelta string                 `json:"account_ram_delta"`
	NetUsage        string                 `json:"net_usage"`
	Elapsed         string                 `json:"elapsed"`
	ErrorCode       interface{}            `json:"error_code"`
	CpuUsageUs      string                 `json:"cpu_usage_us"`
	FailedDtrxTrace interface{}            `json:"failed_dtrx_trace"`
	Except          string                 `json:"except"`
	Status          string                 `json:"status"`
	Id              string                 `json:"id"`
	ActionTraces    []map[string]interface{} `json:"action_traces"`
	//ActionTraces    []struct {
	//	ContextFree          string                 `json:"context_free"`
	//	Act                  map[string]interface{} `json:"act"`
	//	AccountRamDeltas     interface{}            `json:"account_ram_deltas"`
	//	ActionOrdinal        string                 `json:"action_ordinal"`
	//	Elapsed              string                 `json:"elapsed"`
	//	ErrorCode            string                 `json:"error_code"`
	//	Except               string                 `json:"except"`
	//	Receiver             string                 `json:"receiver"`
	//	CreatorActionOrdinal string                 `json:"creator_action_ordinal"`
	//	Receipt              map[string]interface{} `json:"receipt"`
	//	Console              string                 `json:"console"`
	//} `json:"action_traces"`
}

func Trace(b []byte) (trace json.RawMessage, err error) {
	msg := &MsgData{}
	err = json.Unmarshal(b, msg)
	if err != nil || msg.Data == nil {
		return
	}
	tr := &TraceResult{}
	err = json.Unmarshal(msg.Data, tr)
	if err != nil {
		return
	}
	tr.Id = tr.Trace.Id
	tr.BlockNum, _ = strconv.ParseUint(tr.BlockNum.(string), 10, 32)
	tr.RecordType = "trace"
	for _, t := range tr.Trace.ActionTraces {
		// act.data and act.data.owner both can present as a string, maybe it's an ABI problem?
		// but it violates elasticsearch's schema and they won't get indexed if not a struct:
		switch t["act"].(type) {
		case map[string]interface{}:
			switch t["act"].(map[string]interface{})["data"].(type) {
			case string:
				t["act"].(map[string]interface{})["data"] = map[string]string{"raw": t["act"].(map[string]interface{})["data"].(string)}
			case map[string]interface{}:
				switch t["act"].(map[string]interface{})["data"].(map[string]interface{})["owner"].(type) {
				case string:
					t["act"].(map[string]interface{})["data"].(map[string]interface{})["owner"] = map[string]string{"data": t["act"].(map[string]interface{})["data"].(map[string]interface{})["owner"].(string)}
				}
			}
		}
		// trie-search and replace for integer and float casts
		t = Fixup(t)
	}
	return json.Marshal(tr)
}
