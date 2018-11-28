package dnsforward

import (
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/miekg/dns"
)

func RR(rr string) dns.RR {
	r, err := dns.NewRR(rr)
	if err != nil {
		panic(err)
	}
	return r
}

// deepEqual is same as deep.Equal, except:
//  * ignores Id when comparing
//  * question names are not case sensetive
func deepEqualMsg(left *dns.Msg, right *dns.Msg) []string {
	temp := *left
	temp.Id = right.Id
	for i := range left.Question {
		left.Question[i].Name = strings.ToLower(left.Question[i].Name)
	}
	for i := range right.Question {
		right.Question[i].Name = strings.ToLower(right.Question[i].Name)
	}
	return deep.Equal(&temp, right)
}

func TestCacheSanity(t *testing.T) {
	cache := cache{}
	request := dns.Msg{}
	request.SetQuestion("google.com.", dns.TypeA)
	_, ok := cache.Get(&request)
	if ok {
		t.Fatal("empty cache replied with positive response")
	}
}

type tests struct {
	cache []testEntry
	cases []testCase
}

type testEntry struct {
	q string
	t uint16
	a []dns.RR
}

type testCase struct {
	q  string
	t  uint16
	a  []dns.RR
	ok bool
}

func TestCache(t *testing.T) {
	tests := tests{
		cache: []testEntry{
			{q: "google.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}},
		},
		cases: []testCase{
			{q: "google.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}, ok: true},
			{q: "google.com.", t: dns.TypeMX, ok: false},
		},
	}
	runTests(t, tests)
}

func TestCacheMixedCase(t *testing.T) {
	tests := tests{
		cache: []testEntry{
			{q: "gOOgle.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}},
		},
		cases: []testCase{
			{q: "gOOgle.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}, ok: true},
			{q: "google.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}, ok: true},
			{q: "GOOGLE.COM.", t: dns.TypeA, a: []dns.RR{RR("google.com. 3600 IN A 8.8.8.8")}, ok: true},
			{q: "gOOgle.com.", t: dns.TypeMX, ok: false},
			{q: "google.com.", t: dns.TypeMX, ok: false},
			{q: "GOOGLE.COM.", t: dns.TypeMX, ok: false},
		},
	}
	runTests(t, tests)
}

func TestZeroTTL(t *testing.T) {
	tests := tests{
		cache: []testEntry{
			{q: "gOOgle.com.", t: dns.TypeA, a: []dns.RR{RR("google.com. 0 IN A 8.8.8.8")}},
		},
		cases: []testCase{
			{q: "google.com.", t: dns.TypeA, ok: false},
			{q: "google.com.", t: dns.TypeA, ok: false},
			{q: "google.com.", t: dns.TypeA, ok: false},
			{q: "google.com.", t: dns.TypeMX, ok: false},
			{q: "google.com.", t: dns.TypeMX, ok: false},
			{q: "google.com.", t: dns.TypeMX, ok: false},
		},
	}
	runTests(t, tests)
}

func runTests(t *testing.T, tests tests) {
	t.Helper()
	cache := cache{}
	for _, tc := range tests.cache {
		reply := dns.Msg{}
		reply.SetQuestion(tc.q, tc.t)
		reply.Response = true
		reply.Answer = tc.a
		cache.Set(&reply)
	}
	for _, tc := range tests.cases {
		request := dns.Msg{}
		request.SetQuestion(tc.q, tc.t)
		val, ok := cache.Get(&request)
		if diff := deep.Equal(ok, tc.ok); diff != nil {
			t.Error(diff)
		}
		if tc.a != nil {
			if ok == false {
				continue
			}
			reply := dns.Msg{}
			reply.SetQuestion(tc.q, tc.t)
			reply.Response = true
			reply.Answer = tc.a
			cache.Set(&reply)
			if diff := deepEqualMsg(val, &reply); diff != nil {
				t.Error(diff)
			} else {
				if diff := deep.Equal(val, reply); diff == nil {
					t.Error("different message ID were not caught")
				}
			}
		}
	}
}
