package home

import "testing"

func TestClients(t *testing.T) {
	var c Client
	var e error
	var b bool

	clientsInit()

	// add
	c = Client{
		IP:   "1.1.1.1",
		Name: "client1",
	}
	b, e = clientAdd(c)
	if !b || e != nil {
		t.Fatalf("clientAdd #1")
	}

	// add #2
	c = Client{
		IP:   "2.2.2.2",
		Name: "client2",
	}
	b, e = clientAdd(c)
	if !b || e != nil {
		t.Fatalf("clientAdd #2")
	}

	c, b = clientFind("1.1.1.1")
	if !b || c.Name != "client1" {
		t.Fatalf("clientFind #1")
	}

	c, b = clientFind("2.2.2.2")
	if !b || c.Name != "client2" {
		t.Fatalf("clientFind #2")
	}

	// failed add - name in use
	c = Client{
		IP:   "1.2.3.5",
		Name: "client1",
	}
	b, _ = clientAdd(c)
	if b {
		t.Fatalf("clientAdd - name in use")
	}

	// failed add - ip in use
	c = Client{
		IP:   "2.2.2.2",
		Name: "client3",
	}
	b, e = clientAdd(c)
	if b || e == nil {
		t.Fatalf("clientAdd - ip in use")
	}

	// get
	if clientExists("1.2.3.4") {
		t.Fatalf("clientExists")
	}
	if !clientExists("1.1.1.1") {
		t.Fatalf("clientExists #1")
	}
	if !clientExists("2.2.2.2") {
		t.Fatalf("clientExists #2")
	}

	// failed update - no such name
	c.IP = "1.2.3.0"
	c.Name = "client3"
	if clientUpdate("client3", c) == nil {
		t.Fatalf("clientUpdate")
	}

	// failed update - name in use
	c.IP = "1.2.3.0"
	c.Name = "client2"
	if clientUpdate("client1", c) == nil {
		t.Fatalf("clientUpdate - name in use")
	}

	// failed update - ip in use
	c.IP = "2.2.2.2"
	c.Name = "client1"
	if clientUpdate("client1", c) == nil {
		t.Fatalf("clientUpdate - ip in use")
	}

	// update
	c.IP = "1.1.1.2"
	c.Name = "client1"
	if clientUpdate("client1", c) != nil {
		t.Fatalf("clientUpdate")
	}

	// get after update
	if clientExists("1.1.1.1") || !clientExists("1.1.1.2") {
		t.Fatalf("clientExists - get after update")
	}

	// failed remove - no such name
	if clientDel("client3") {
		t.Fatalf("clientDel - no such name")
	}

	// remove
	if !clientDel("client1") || clientExists("1.1.1.2") {
		t.Fatalf("clientDel")
	}

	// add host client
	b, e = clientAddHost("1.1.1.1", "host", ClientSourceARP)
	if !b || e != nil {
		t.Fatalf("clientAddHost")
	}

	// failed add - ip exists
	b, e = clientAddHost("1.1.1.1", "host1", ClientSourceRDNS)
	if b || e != nil {
		t.Fatalf("clientAddHost - ip exists")
	}

	// overwrite with new data
	b, e = clientAddHost("1.1.1.1", "host2", ClientSourceARP)
	if !b || e != nil {
		t.Fatalf("clientAddHost - overwrite with new data")
	}

	// overwrite with new data (higher priority)
	b, e = clientAddHost("1.1.1.1", "host3", ClientSourceHostsFile)
	if !b || e != nil {
		t.Fatalf("clientAddHost - overwrite with new data (higher priority)")
	}

	// get
	if !clientExists("1.1.1.1") {
		t.Fatalf("clientAddHost")
	}
}
