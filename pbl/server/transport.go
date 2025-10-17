package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/raft"
)

// HTTPTransport implementa a interface raft.Transport sobre HTTP.
type HTTPTransport struct {
	localAddr raft.ServerAddress
	rpcChan   chan raft.RPC
	client    *http.Client
}

// NewHTTPTransport cria um novo transporte.
func NewHTTPTransport(localAddr raft.ServerAddress) *HTTPTransport {
	return &HTTPTransport{
		localAddr: localAddr,
		rpcChan:   make(chan raft.RPC),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// ---- Funções da Interface raft.Transport ----

func (t *HTTPTransport) LocalAddr() raft.ServerAddress {
	return t.localAddr
}

func (t *HTTPTransport) AppendEntries(id raft.ServerID, target raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
	return t.sendRPC(target, "AppendEntries", args, resp)
}

func (t *HTTPTransport) RequestVote(id raft.ServerID, target raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	return t.sendRPC(target, "RequestVote", args, resp)
}

func (t *HTTPTransport) AppendEntriesPipeline(id raft.ServerID, target raft.ServerAddress) (raft.AppendPipeline, error) {
	return nil, raft.ErrPipelineReplicationNotSupported
}

// TimeoutNow é a nova função que substitui AppendEntriesPipeline
func (t *HTTPTransport) TimeoutNow(id raft.ServerID, target raft.ServerAddress, args *raft.TimeoutNowRequest, resp *raft.TimeoutNowResponse) error {
	return t.sendRPC(target, "TimeoutNow", args, resp)
}


func (t *HTTPTransport) InstallSnapshot(id raft.ServerID, target raft.ServerAddress, args *raft.InstallSnapshotRequest, resp *raft.InstallSnapshotResponse, data io.Reader) error {
	return fmt.Errorf("InstallSnapshot not implemented")
}

func (t *HTTPTransport) EncodePeer(id raft.ServerID, addr raft.ServerAddress) []byte {
	return []byte(addr)
}

func (t *HTTPTransport) DecodePeer(buf []byte) raft.ServerAddress {
	return raft.ServerAddress(string(buf))
}

func (t *HTTPTransport) Consumer() <-chan raft.RPC {
	return t.rpcChan
}

func (t *HTTPTransport) SetHeartbeatHandler(cb func(rpc raft.RPC)) {
	// No-op. Todas as RPCs são tratadas da mesma forma.
}

// ---- Funções Auxiliares ----

func (t *HTTPTransport) sendRPC(target raft.ServerAddress, rpcType string, args interface{}, resp interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(args); err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/raft", target)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("X-Raft-RPC-Type", rpcType)

	httpResp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("rpc to %s failed with status: %d", target, httpResp.StatusCode)
	}

	if err := gob.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return err
	}
	return nil
}

func (t *HTTPTransport) HandleRaftRequest(w http.ResponseWriter, r *http.Request) {
	var req interface{}
	rpcType := r.Header.Get("X-Raft-RPC-Type")

	switch rpcType {
	case "AppendEntries":
		req = &raft.AppendEntriesRequest{}
	case "RequestVote":
		req = &raft.RequestVoteRequest{}
	case "TimeoutNow":
		req = &raft.TimeoutNowRequest{}
	default:
		log.Printf("ERROR: invalid rpc type: %s", rpcType)
		http.Error(w, "invalid rpc type", http.StatusBadRequest)
		return
	}

	if err := gob.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respChan := make(chan raft.RPCResponse, 1)
	rpc := raft.RPC{
		Command:  req,
		RespChan: respChan,
	}
	t.rpcChan <- rpc

	rpcResp := <-respChan
	if rpcResp.Error != nil {
		http.Error(w, rpcResp.Error.Error(), http.StatusInternalServerError)
		return
	}

	if err := gob.NewEncoder(w).Encode(rpcResp.Response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}