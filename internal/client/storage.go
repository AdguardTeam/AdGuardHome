package client

import (
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// Config is the client storage configuration structure.
//
// TODO(s.chzhen):  Expand.
type Config struct {
	// AllowedTags is a list of all allowed client tags.
	AllowedTags []string
}

// Storage contains information about persistent and runtime clients.
type Storage struct {
	// allowedTags is a set of all allowed tags.
	allowedTags *container.MapSet[string]

	// mu protects indexes of persistent and runtime clients.
	mu *sync.Mutex

	// index contains information about persistent clients.
	index *index

	// runtimeIndex contains information about runtime clients.
	//
	// TODO(s.chzhen):  Use it.
	runtimeIndex *RuntimeIndex
}

// NewStorage returns initialized client storage.  conf must not be nil.
func NewStorage(conf *Config) (s *Storage) {
	allowedTags := container.NewMapSet(conf.AllowedTags...)

	return &Storage{
		allowedTags:  allowedTags,
		mu:           &sync.Mutex{},
		index:        newIndex(),
		runtimeIndex: NewRuntimeIndex(),
	}
}

// Add stores persistent client information or returns an error.
func (s *Storage) Add(p *Persistent) (err error) {
	defer func() { err = errors.Annotate(err, "adding client: %w") }()

	err = p.validate(s.allowedTags)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.index.clashesUID(p)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	err = s.index.clashes(p)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	s.index.add(p)

	log.Debug("client storage: added %q: IDs: %q [%d]", p.Name, p.IDs(), s.index.size())

	return nil
}

// FindByName finds persistent client by name.  And returns its shallow copy.
func (s *Storage) FindByName(name string) (p *Persistent, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok = s.index.findByName(name)
	if ok {
		return p.ShallowClone(), ok
	}

	return nil, false
}

// Find finds persistent client by string representation of the client ID, IP
// address, or MAC.  And returns its shallow copy.
func (s *Storage) Find(id string) (p *Persistent, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok = s.index.find(id)
	if ok {
		return p.ShallowClone(), ok
	}

	return nil, false
}

// FindLoose is like [Storage.Find] but it also tries to find a persistent
// client by IP address without zone.  It strips the IPv6 zone index from the
// stored IP addresses before comparing, because querylog entries don't have it.
// See TODO on [querylog.logEntry.IP].
//
// Note that multiple clients can have the same IP address with different zones.
// Therefore, the result of this method is indeterminate.
func (s *Storage) FindLoose(ip netip.Addr, id string) (p *Persistent, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok = s.index.find(id)
	if ok {
		return p.ShallowClone(), ok
	}

	p = s.index.findByIPWithoutZone(ip)
	if p != nil {
		return p.ShallowClone(), true
	}

	return nil, false
}

// FindByMAC finds persistent client by MAC and returns its shallow copy.
func (s *Storage) FindByMAC(mac net.HardwareAddr) (p *Persistent, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok = s.index.findByMAC(mac)
	if ok {
		return p.ShallowClone(), ok
	}

	return nil, false
}

// RemoveByName removes persistent client information.  ok is false if no such
// client exists by that name.
func (s *Storage) RemoveByName(name string) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.index.findByName(name)
	if !ok {
		return false
	}

	if err := p.CloseUpstreams(); err != nil {
		log.Error("client storage: removing client %q: %s", p.Name, err)
	}

	s.index.remove(p)

	return true
}

// Update finds the stored persistent client by its name and updates its
// information from p.
func (s *Storage) Update(name string, p *Persistent) (err error) {
	defer func() { err = errors.Annotate(err, "updating client: %w") }()

	err = p.validate(s.allowedTags)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.index.findByName(name)
	if !ok {
		return fmt.Errorf("client %q is not found", name)
	}

	// Client p has a newly generated UID, so replace it with the stored one.
	//
	// TODO(s.chzhen):  Remove when frontend starts handling UIDs.
	p.UID = stored.UID

	err = s.index.clashes(p)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	s.index.remove(stored)
	s.index.add(p)

	return nil
}

// RangeByName calls f for each persistent client sorted by name, unless cont is
// false.
func (s *Storage) RangeByName(f func(c *Persistent) (cont bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index.rangeByName(f)
}

// Size returns the number of persistent clients.
func (s *Storage) Size() (n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.index.size()
}

// CloseUpstreams closes upstream configurations of persistent clients.
func (s *Storage) CloseUpstreams() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.index.closeUpstreams()
}

// ClientRuntime returns a copy of the saved runtime client by ip.  If no such
// client exists, returns nil.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) ClientRuntime(ip netip.Addr) (rc *Runtime) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runtimeIndex.Client(ip)
}

// AddRuntime saves the runtime client information in the storage.  IP address
// of a client must be unique.  rc must not be nil.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) AddRuntime(rc *Runtime) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runtimeIndex.Add(rc)
}

// SizeRuntime returns the number of the runtime clients.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) SizeRuntime() (n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runtimeIndex.Size()
}

// RangeRuntime calls f for each runtime client in an undefined order.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) RangeRuntime(f func(rc *Runtime) (cont bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runtimeIndex.Range(f)
}

// DeleteRuntime removes the runtime client by ip.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) DeleteRuntime(ip netip.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runtimeIndex.Delete(ip)
}

// DeleteBySource removes all runtime clients that have information only from
// the specified source and returns the number of removed clients.
//
// TODO(s.chzhen):  Use it.
func (s *Storage) DeleteBySource(src Source) (n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runtimeIndex.DeleteBySource(src)
}
