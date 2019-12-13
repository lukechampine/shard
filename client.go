package shard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/NebulousLabs/Sia/crypto"
	"gitlab.com/NebulousLabs/Sia/encoding"
	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/types"
	"lukechampine.com/us/hostdb"
)

// A Client communicates with a SHARD server.
type Client struct {
	addr string
}

func (c *Client) req(route string, fn func(*http.Response) error) error {
	resp, err := http.Get(fmt.Sprintf("%v%v", c.addr, route))
	if err != nil {
		return err
	}
	defer io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()

	if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
		errString, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(errString))
	}
	return fn(resp)
}

// ChainHeight returns the current block height.
func (c *Client) ChainHeight() (types.BlockHeight, error) {
	var height types.BlockHeight
	err := c.req("/height", func(resp *http.Response) error {
		return json.NewDecoder(resp.Body).Decode(&height)
	})
	return height, err
}

// Synced returns whether the SHARD server is synced.
func (c *Client) Synced() (bool, error) {
	var synced bool
	err := c.req("/synced", func(resp *http.Response) error {
		data, err := ioutil.ReadAll(io.LimitReader(resp.Body, 8))
		if err != nil {
			return err
		}
		synced, err = strconv.ParseBool(string(data))
		return err
	})
	return synced, err
}

// ResolveHostKey resolves a host public key to that host's most recently
// announced network address.
func (c *Client) ResolveHostKey(pubkey hostdb.HostPublicKey) (modules.NetAddress, error) {
	var ha modules.HostAnnouncement
	var sig crypto.Signature
	err := c.req("/host/"+string(pubkey), func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNoContent {
			return errors.New("no record of that host")
		} else if resp.StatusCode == http.StatusGone {
			return errors.New("ambiguous pubkey")
		}
		return encoding.NewDecoder(resp.Body, encoding.DefaultAllocLimit).DecodeAll(&ha, &sig)
	})
	if err != nil {
		return "", err
	}
	if !pubkey.VerifyHash(crypto.HashObject(ha), sig[:]) {
		return "", errors.New("invalid signature")
	}
	return ha.NetAddress, err
}

// LookupHost returns the host public key matching the specified prefix.
func (c *Client) LookupHost(prefix string) (hostdb.HostPublicKey, error) {
	if !strings.HasPrefix(prefix, "ed25519:") {
		prefix = "ed25519:" + prefix
	}
	var ha modules.HostAnnouncement
	var sig crypto.Signature
	err := c.req("/host/"+prefix, func(resp *http.Response) error {
		if resp.ContentLength == 0 {
			return errors.New("no record of that host")
		}
		return encoding.NewDecoder(resp.Body, encoding.DefaultAllocLimit).DecodeAll(&ha, &sig)
	})
	if err != nil {
		return "", err
	}
	return hostdb.HostKeyFromSiaPublicKey(ha.PublicKey), nil
}

// NewClient returns a Client that communicates with the SHARD
// server at the specified address.
func NewClient(addr string) *Client {
	// use https by default
	if !strings.HasPrefix(addr, "https://") && !strings.HasPrefix(addr, "http://") {
		addr = "https://" + addr
	}
	return &Client{addr: addr}
}
