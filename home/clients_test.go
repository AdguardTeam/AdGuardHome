package home

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"

	"github.com/stretchr/testify/assert"
)

func TestClients(t *testing.T) {
	var c Client
	var e error
	var b bool
	clients := clientsContainer{}
	clients.testing = true

	clients.Init(nil, nil, nil)

	// add
	c = Client{
		IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
		Name: "client1",
	}
	b, e = clients.Add(c)
	if !b || e != nil {
		t.Fatalf("Add #1")
	}

	// add #2
	c = Client{
		IDs:  []string{"2.2.2.2"},
		Name: "client2",
	}
	b, e = clients.Add(c)
	if !b || e != nil {
		t.Fatalf("Add #2")
	}

	c, b = clients.Find("1.1.1.1")
	assert.True(t, b && c.Name == "client1")

	c, b = clients.Find("1:2:3::4")
	assert.True(t, b && c.Name == "client1")

	c, b = clients.Find("2.2.2.2")
	assert.True(t, b && c.Name == "client2")

	// failed add - name in use
	c = Client{
		IDs:  []string{"1.2.3.5"},
		Name: "client1",
	}
	b, _ = clients.Add(c)
	if b {
		t.Fatalf("Add - name in use")
	}

	// failed add - ip in use
	c = Client{
		IDs:  []string{"2.2.2.2"},
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
	c.IDs = []string{"1.2.3.0"}
	c.Name = "client3"
	if clients.Update("client3", c) == nil {
		t.Fatalf("Update")
	}

	// failed update - name in use
	c.IDs = []string{"1.2.3.0"}
	c.Name = "client2"
	if clients.Update("client1", c) == nil {
		t.Fatalf("Update - name in use")
	}

	// failed update - ip in use
	c.IDs = []string{"2.2.2.2"}
	c.Name = "client1"
	if clients.Update("client1", c) == nil {
		t.Fatalf("Update - ip in use")
	}

	// update
	c.IDs = []string{"1.1.1.2"}
	c.Name = "client1"
	if clients.Update("client1", c) != nil {
		t.Fatalf("Update")
	}

	// get after update
	assert.True(t, !clients.Exists("1.1.1.1", ClientSourceHostsFile))
	assert.True(t, clients.Exists("1.1.1.2", ClientSourceHostsFile))

	// update - rename
	c.IDs = []string{"1.1.1.2"}
	c.Name = "client1-renamed"
	c.UseOwnSettings = true
	assert.True(t, clients.Update("client1", c) == nil)
	c = Client{}
	c, b = clients.Find("1.1.1.2")
	assert.True(t, b && c.Name == "client1-renamed" && c.IDs[0] == "1.1.1.2" && c.UseOwnSettings)
	assert.True(t, clients.list["client1"] == nil)

	// failed remove - no such name
	if clients.Del("client3") {
		t.Fatalf("Del - no such name")
	}

	// remove
	assert.True(t, !(!clients.Del("client1-renamed") || clients.Exists("1.1.1.2", ClientSourceHostsFile)))

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

func TestClientsWhois(t *testing.T) {
	var c Client
	clients := clientsContainer{}
	clients.testing = true
	clients.Init(nil, nil, nil)

	whois := [][]string{{"orgname", "orgname-val"}, {"country", "country-val"}}
	// set whois info on new client
	clients.SetWhoisInfo("1.1.1.255", whois)
	assert.True(t, clients.ipHost["1.1.1.255"].WhoisInfo[0][1] == "orgname-val")

	// set whois info on existing auto-client
	_, _ = clients.AddHost("1.1.1.1", "host", ClientSourceRDNS)
	clients.SetWhoisInfo("1.1.1.1", whois)
	assert.True(t, clients.ipHost["1.1.1.1"].WhoisInfo[0][1] == "orgname-val")

	// Check that we cannot set whois info on a manually-added client
	c = Client{
		IDs:  []string{"1.1.1.2"},
		Name: "client1",
	}
	_, _ = clients.Add(c)
	clients.SetWhoisInfo("1.1.1.2", whois)
	assert.True(t, clients.ipHost["1.1.1.2"] == nil)
	_ = clients.Del("client1")
}

func TestClientsAddExisting(t *testing.T) {
	var c Client
	clients := clientsContainer{}
	clients.testing = true
	clients.Init(nil, nil, nil)

	// some test variables
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	testIP := "1.2.3.4"

	// add a client
	c = Client{
		IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa", "2.2.2.0/24"},
		Name: "client1",
	}
	ok, err := clients.Add(c)
	assert.True(t, ok)
	assert.Nil(t, err)

	// add an auto-client with the same IP - it's allowed
	ok, err = clients.AddHost("1.1.1.1", "test", ClientSourceRDNS)
	assert.True(t, ok)
	assert.Nil(t, err)

	// now some more complicated stuff
	// first, init a DHCP server with a single static lease
	config := dhcpd.ServerConfig{
		DBFilePath: "leases.db",
	}
	defer func() { _ = os.Remove("leases.db") }()
	clients.dhcpServer = dhcpd.Create(config)
	err = clients.dhcpServer.AddStaticLease(dhcpd.Lease{
		HWAddr:   mac,
		IP:       net.ParseIP(testIP).To4(),
		Hostname: "testhost",
		Expiry:   time.Now().Add(time.Hour),
	})
	assert.Nil(t, err)

	// add a new client with the same IP as for a client with MAC
	c = Client{
		IDs:  []string{testIP},
		Name: "client2",
	}
	ok, err = clients.Add(c)
	assert.True(t, ok)
	assert.Nil(t, err)

	// add a new client with the IP from the client1's IP range
	c = Client{
		IDs:  []string{"2.2.2.2"},
		Name: "client3",
	}
	ok, err = clients.Add(c)
	assert.True(t, ok)
	assert.Nil(t, err)
}

func TestClientsCustomUpstream(t *testing.T) {
	clients := clientsContainer{}
	clients.testing = true

	clients.Init(nil, nil, nil)

	// add client with upstreams
	client := Client{
		IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
		Name: "client1",
		Upstreams: []string{
			"1.1.1.1",
			"[/example.org/]8.8.8.8",
		},
	}
	ok, err := clients.Add(client)
	assert.Nil(t, err)
	assert.True(t, ok)

	config := clients.FindUpstreams("1.2.3.4")
	assert.Nil(t, config)

	config = clients.FindUpstreams("1.1.1.1")
	assert.NotNil(t, config)
	assert.Equal(t, 1, len(config.Upstreams))
	assert.Equal(t, 1, len(config.DomainReservedUpstreams))
}
