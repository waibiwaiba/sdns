package doh

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/miekg/dns"
)

const minMsgHeaderSize = 12

// HandleWireFormat handle wire format
func HandleWireFormat(handle func(*dns.Msg) *dns.Msg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			buf []byte
			err error
		)

		switch r.Method {
		case http.MethodGet:
			buf, err = base64.RawURLEncoding.DecodeString(r.URL.Query().Get("dns"))
			if len(buf) == 0 || err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		case http.MethodPost:
			if r.Header.Get("Content-Type") != "application/dns-message" {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				return
			}

			buf, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if len(buf) < minMsgHeaderSize {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		req := new(dns.Msg)
		if err := req.Unpack(buf); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		msg := handle(req)
		if msg == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// 创建一个新的DNS查询请求，查询baidu.com.
		reqBaidu := new(dns.Msg)
		reqBaidu.SetQuestion("baidu.com.", dns.TypeA) // 假设我们查询A记录

		// 使用同样的handle函数处理这个新的查询
		msgBaidu := handle(reqBaidu)
		if msgBaidu == nil {
			// 处理错误，可以根据需要决定是否要终止处理
			http.Error(w, "Failed to query baidu.com.", http.StatusInternalServerError)
			return
		}

		// 将baidu.com的查询结果添加到原始响应中
		msg.Answer = append(msg.Answer, msgBaidu.Answer...)

		// 这里继续你的响应处理，例如，将msg序列化并写入响应体

		packed, err := msg.Pack()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/dns-message")

		_, _ = w.Write(packed)
	}
}

// HandleJSON handle json format
func HandleJSON(handle func(*dns.Msg) *dns.Msg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		name = dns.Fqdn(name)

		qtype := ParseQTYPE(r.URL.Query().Get("type"))
		if qtype == dns.TypeNone {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		req := new(dns.Msg)
		req.SetQuestion(name, qtype)
		req.AuthenticatedData = true

		if r.URL.Query().Get("cd") == "true" {
			req.CheckingDisabled = true
		}

		opt := &dns.OPT{
			Hdr: dns.RR_Header{
				Name:   ".",
				Class:  dns.DefaultMsgSize,
				Rrtype: dns.TypeOPT,
			},
		}

		if r.URL.Query().Get("do") == "true" {
			opt.SetDo()
		}

		req.Extra = append(req.Extra, opt)

		msg := handle(req)
		if msg == nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		json, err := json.Marshal(NewMsg(msg))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.Header().Set("Content-Type", "application/x-javascript")
		} else {
			w.Header().Set("Content-Type", "application/dns-json")
		}

		_, _ = w.Write(json)
	}
}
