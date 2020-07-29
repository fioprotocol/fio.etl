package transform

import (
	"encoding/binary"
	"fmt"
	"github.com/importcjj/trie-go"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var intTrie, floatTrie, boolTrie *trie.Trie

// Fixup applies type casting for fields that need changed. For example an amount that should be an int but is a string
func Fixup(fixme map[string]interface{}) map[string]interface{} {
	for _, t := range []string{"int", "bool", "float"} {
		seekFor(fixme, nil, t)
	}
	return fixme
}

// seekFor is a recurisve function that traverses the trie looking for fields to cast
func seekFor(target map[string]interface{}, leaf []string, kind string) {
	// for tests, if init() not run ....
	if intTrie == nil || boolTrie == nil || floatTrie == nil {
		intTrie, floatTrie, boolTrie = BuildTrie()
	}
	if intTrie == nil || boolTrie == nil || floatTrie == nil {
		log.Fatal("trie nil!")
	}
	if leaf == nil {
		leaf = make([]string, 0)
	}
	for k := range target {
		switch target[k].(type) {
		case nil:
			return
		case map[string]interface{}:
			leaf = append(leaf, k)
			seekFor(target[k].(map[string]interface{}), leaf, kind)
		default:
			s := strings.Join(leaf, "/")
			if strings.Contains(s, "//") {
				break
			}
			fmt.Println(leaf, kind, "/"+s)
			switch kind {
			case "int":
				if intTrie != nil && intTrie.Has("/" + s) {
					target[k] = toInt(target[k])
				}
			case "float":
				if floatTrie.Has("/" + s) {
					target[k] = toFloat(target[k])
				}
			case "bool":
				if boolTrie.Has("/" + s) {
					target[k] = toBool(target[k])
				}
			}
		}
	}
}

var (
	stripNotDigit = regexp.MustCompile(`(\d+)\D*?`) // capture the first group of consecutive numbers, will chop a float
	onlyDigit     = regexp.MustCompile(`^\d$`)
	isAsset       = regexp.MustCompile(` FIO$`)
)

func toInt(v interface{}) interface{} {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		// nothing to see here... move along.
		return v
	case string:
		if !onlyDigit.MatchString(v.(string)) {
			v = stripNotDigit.FindString(v.(string))
		}
		v, _ = strconv.ParseInt(v.(string), 10, 64)
		return v
	case []byte:
		msb := make([]byte, 8-len(v.([]byte)))
		v = binary.LittleEndian.Uint64(append(v.([]byte), msb...))
		return v
	case float32, float64:
		v = int64(math.Round(v.(float64)))
		return v
	}
	v = int64(0)
	return v
}

func toBool(v interface{}) interface{} {
	switch v.(type) {
	case bool:
		return v
	case string:
		v, _ = strconv.ParseBool(v.(string))
		return v
	}
	v = false
	return v
}

func toFloat(v interface{}) interface{} {
	switch v.(type) {
	case float64, float32:
		return v
	case string:
		if isAsset.MatchString(v.(string)) {
			v, _ = strconv.ParseFloat(strings.Split(v.(string), " ")[0], 64)
			return v
		}
		v, _ = strconv.ParseFloat(v.(string), 64)
		return v
	case int, int8, int16, int32, int64, uint8, uint16, uint32:
		v = float64(v.(int64))
	case uint64:
		if v.(uint64) > uint64(math.Round(math.MaxFloat64)) {
			v = math.MaxFloat64
		}
		v = float64(v.(uint64))
	}
	return v
}

func BuildTrie() (intTrie *trie.Trie, floatTrie *trie.Trie, boolTrie *trie.Trie) {
	intTrie = trie.New()
	_ = intTrie.Put("/", true)
	floatTrie = trie.New()
	_ = floatTrie.Put("/", true)
	boolTrie = trie.New()
	_ = boolTrie.Put("/", true)

	var err error
	mkTrie := func(leafs []string, t *trie.Trie) {
		for _, row := range leafs {
			leaf := strings.Split(row, ".")
			for i := range leaf {
				if !boolTrie.Has("/" + strings.Join(leaf[:i+1], "/")) {
					err = boolTrie.Put("/"+strings.Join(leaf[:i+1], "/"), true)
					if err != nil {
						elog.Fatal(err)
					}
				}
			}
		}
	}
	mkTrie(wantBool, boolTrie)
	mkTrie(wantFloat, floatTrie)
	mkTrie(wantInt, intTrie)
	if intTrie == nil || boolTrie == nil || floatTrie == nil {
		log.Fatal("could not init tries")
	}
	return
}

// for now only fixing up traces, but flexible enough for generic work:
var (
	wantBool = []string{
		`trace.scheduled`,
	}
	wantFloat = []string{
		`trace.action_traces.receipt.act.data.quantity`,
	}
	wantInt = []string{
		`trace.account_ram_delta`,
		`trace.action_traces.account_ram_deltas.delta`,
		`trace.action_traces.act.data.amount`,
		`trace.action_traces.act.data.max_fee`,
		`trace.action_traces.act.data.quantity`,
		`trace.action_traces.act.data.suf_amount`,
		`trace.action_traces.action_ordinal`,
		`trace.action_traces.creator_action_ordinal`,
		`trace.action_traces.elapsed`,
		`trace.action_traces.receipt.abi_sequence`,
		`trace.action_traces.receipt.act.data.amount`,
		`trace.action_traces.receipt.act.data.max_fee`,
		`trace.action_traces.receipt.act.data.quantity`,
		`trace.action_traces.receipt.act.data.suf_amount`,
		`trace.action_traces.receipt.auth_sequence.sequence`,
		`trace.action_traces.receipt.code_sequence`,
		`trace.action_traces.receipt.global_sequence`,
		`trace.action_traces.receipt.recv_sequence`,
		`trace.cpu_usage_us`,
		`trace.elapsed`,
		`trace.net_usage_words`,
		`trace.net_usage`,
		`trace.partial.delay_sec`,
		`trace.partial.max_cpu_usage_ms`,
		`trace.partial.max_net_usage_words`,
		`trace.partial.ref_block_num`,
		`trace.partial.ref_block_prefix`,
	}
)