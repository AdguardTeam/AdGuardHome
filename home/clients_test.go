package home

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClients(t *testing.T) {
	var c Client
	var e error
	var b bool
	clients := clientsContainer{}

	clients.Init()

	// add
	c = Client{
		IP:   "1.1.1.1",
		Name: "client1",
	}
	b, e = clients.Add(c)
	if !b || e != nil {
		t.Fatalf("Add #1")
	}

	// add #2
	c = Client{
		IP:   "2.2.2.2",
		Name: "client2",
	}
	b, e = clients.Add(c)
	if !b || e != nil {
		t.Fatalf("Add #2")
	}

	c, b = clients.Find("1.1.1.1")
	if !b || c.Name != "client1" {
		t.Fatalf("Find #1")
	}

	c, b = clients.Find("2.2.2.2")
	if !b || c.Name != "client2" {
		t.Fatalf("Find #2")
	}

	// failed add - name in use
	c = Client{
		IP:   "1.2.3.5",
		Name: "client1",
	}
	b, _ = clients.Add(c)
	if b {
		t.Fatalf("Add - name in use")
	}

	// failed add - ip in use
	c = Client{
		IP:   "2.2.2.2",
		Name: "client3",
	}
	b, e = clients.Add(c)
	if b || e == nil {
		t.Fatalf("Add - ip in use")
	}

	// get
	assert.True(t, !clients.Exists("1.2.3.4", ClientSourceHostsFile))
	assert.True(t, clients.Exists("1.1.1.1", ClientSourceHostsFile))
	assert.True(t, clients.Exists("2.2.2.2", ClientSourceHostsFile))

	// failed update - no such name
	c.IP = "1.2.3.0"
	c.Name = "client3"
	if clients.Update("client3", c) == nil {
		t.Fatalf("Update")
	}

	// failed update - name in use
	c.IP = "1.2.3.0"
	c.Name = "client2"
	if clients.Update("client1", c) == nil {
		t.Fatalf("Update - name in use")
	}

	// failed update - ip in use
	c.IP = "2.2.2.2"
	c.Name = "client1"
	if clients.Update("client1", c) == nil {
		t.Fatalf("Update - ip in use")
	}

	// update
	c.IP = "1.1.1.2"
	c.Name = "client1"
	if clients.Update("client1", c) != nil {
		t.Fatalf("Update")
	}

	// get after update
	assert.True(t, !(clients.Exists("1.1.1.1", ClientSourceHostsFile) || !clients.Exists("1.1.1.2", ClientSourceHostsFile)))

	// failed remove - no such name
	if clients.Del("client3") {
		t.Fatalf("Del - no such name")
	}

	// remove
	assert.True(t, !(!clients.Del("client1") || clients.Exists("1.1.1.2", ClientSourceHostsFile)))

	// add host client
	b, e = clients.AddHost("1.1.1.1", "host", ClientSourceARP)
	if !b || e != nil {
		t.Fatalf("clientAddHost")
	}

	// failed add - ip exists
	b, e = clients.AddHost("1.1.1.1", "host1", ClientSourceRDNS)
	if b || e != nil {
		t.Fatalf("clientAddHost - ip exists")
	}

	// overwrite with new data
	b, e = clients.AddHost("1.1.1.1", "host2", ClientSourceARP)
	if !b || e != nil {
		t.Fatalf("clientAddHost - overwrite with new data")
	}

	// overwrite with new data (higher priority)
	b, e = clients.AddHost("1.1.1.1", "host3", ClientSourceHostsFile)
	if !b || e != nil {
		t.Fatalf("clientAddHost - overwrite with new data (higher priority)")
	}

	// get
	assert.True(t, clients.Exists("1.1.1.1", ClientSourceHostsFile))
}
