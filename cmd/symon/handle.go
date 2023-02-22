package main

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/midbel/symon/proc"
)

func handleStatus(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		return mon.About(), nil
	}
	return handle(fn)
}

type ProcInfo struct {
	Pid    int    `json:"pid"`
	Cmd    string `json:"command"`
	Status string `json:"state"`
	User   string `json:"user"`
	Group  string `json:"group"`
}

func convertProcInfo(info proc.ProcInfo) ProcInfo {
	return ProcInfo{
		Pid:    info.Pid,
		Cmd:    info.Cmd,
		Status: info.Status,
		User:   info.User,
		Group:  info.Group,
	}
}

func handleProcess(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		var (
			list = mon.Process()
			res  = make([]ProcInfo, 0, len(list))
		)
		for i := range list {
			res = append(res, convertProcInfo(list[i]))
		}
		return res, nil
	}
	return handle(fn)
}

type MemInfo struct {
	Total int64 `json:"total`
	Used  int64 `json:"used`
}

func convertMemInfo(info proc.MemInfo) MemInfo {
	return MemInfo{
		Total: info.Total,
		Used:  info.Used(),
	}
}

func handleFree(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		syst, swap := mon.Free()
		list := struct {
			System MemInfo `json:"system"`
			Swap   MemInfo `json:"swap"`
		}{
			System: convertMemInfo(syst),
			Swap:   convertMemInfo(swap),
		}
		return list, nil
	}
	return handle(fn)
}

func handleLoadAvg(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		list := mon.LoadAvg()
		return list, nil
	}
	return handle(fn)
}

type UserInfo struct {
	Type string     `json:"session"`
	User string     `json:"user"`
	Host string     `json:"host"`
	When time.Time  `json:"time"`
	Addr netip.Addr `json:"addr`
}

func convertWho(info proc.Who) UserInfo {
	return UserInfo{
		User: info.User,
		Host: info.Host,
		When: info.When,
		Addr: info.Addr,
	}
}

func handleUsers(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		var (
			list = mon.Users()
			res  = make([]UserInfo, 0, len(list))
		)
		for i := range list {
			res = append(res, convertWho(list[i]))
		}
		return res, nil
	}
	return handle(fn)
}

type ConnInfo struct {
	Local  netip.AddrPort `json:"local"`
	Remote netip.AddrPort `json:"remote"`
	User   string         `json:"user"`
	Proto  string         `json:"protocol"`
	State  string         `json:"state"`
}

func convertConnInfo(info proc.ConnInfo) ConnInfo {
	return ConnInfo{
		Local:  info.Local,
		Remote: info.Remote,
		User:   info.User,
		State:  info.State.String(),
		Proto:  info.Proto,
	}
}

func handleNetstat(mon *Collector) http.Handler {
	fn := func(r *http.Request) (interface{}, error) {
		var (
			list  []proc.ConnInfo
			res   []ConnInfo
			query = r.URL.Query()
		)
		switch strings.ToLower(query.Get("proto")) {
		case "udp":
		case "tcp":
		default:
			list = mon.Conns()
		}
		for i := range list {
			res = append(res, convertConnInfo(list[i]))
		}
		return res, nil
	}
	return handle(fn)
}

type handler func(r *http.Request) (interface{}, error)

func handle(h handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		data, err := h(r)

		w.Header().Set("content-type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(data)
	}
	return http.HandlerFunc(fn)
}
