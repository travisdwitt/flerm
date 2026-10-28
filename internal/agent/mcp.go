package agent

import (
	"bufio"
	"encoding/json"
	"io"
	"slices"
	"strings"
)

type rpcReq struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
		Version   string         `json:"protocolVersion"`
	} `json:"params"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

func opNames() []string {
	names := make([]string, 0, len(Ops))
	for n := range Ops {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

func tools() []map[string]any {
	var out []map[string]any
	for _, n := range opNames() {
		op := Ops[n]
		props := map[string]any{}
		for _, k := range append(slices.Clone(op.Pos), op.Opts...) {
			props[k] = map[string]any{"type": "string"}
		}
		out = append(out, map[string]any{
			"name":        strings.ReplaceAll(n, "-", "_"),
			"description": op.Desc + " Required: " + strings.Join(op.Pos, ", ") + ".",
			"inputSchema": map[string]any{"type": "object", "properties": props, "required": op.Pos},
		})
	}
	return out
}

func reply(msg string, isErr bool) map[string]any {
	return map[string]any{"isError": isErr, "content": []any{map[string]any{"type": "text", "text": msg}}}
}

func call(name string, args map[string]any) map[string]any {
	op, ok := Ops[strings.ReplaceAll(name, "_", "-")]
	if !ok {
		return reply("unknown tool "+name, true)
	}
	p := map[string]string{}
	for k, v := range args {
		p[k] = toStr(v)
	}
	for _, k := range op.Pos {
		if _, ok := p[k]; !ok {
			return reply("missing "+k, true)
		}
	}
	out, err := op.Run(p)
	if err != nil {
		return reply(strings.TrimSpace(out+"\nerror: "+err.Error()), true)
	}
	return reply(out, false)
}

func Serve(r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	enc := json.NewEncoder(w)
	for sc.Scan() {
		var req rpcReq
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil || len(req.ID) == 0 {
			continue
		}
		resp := rpcResp{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			version := req.Params.Version
			if version == "" {
				version = "2024-11-05"
			}
			resp.Result = map[string]any{
				"protocolVersion": version,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "flerm", "version": "1"},
			}
		case "tools/list":
			resp.Result = map[string]any{"tools": tools()}
		case "tools/call":
			resp.Result = call(req.Params.Name, req.Params.Arguments)
		case "ping":
			resp.Result = map[string]any{}
		default:
			resp.Error = &rpcErr{-32601, "method not found: " + req.Method}
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return sc.Err()
}
