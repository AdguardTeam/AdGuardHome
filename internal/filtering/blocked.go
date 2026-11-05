package filtering

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/AdguardTeam/urlfilter/rules"
)

// serviceRules maps a service ID to its filtering rules.
var serviceRules map[string][]*rules.NetworkRule

// serviceIDs contains service IDs sorted alphabetically.
var serviceIDs []string

// initBlockedServices initializes package-level blocked service data.  l must
// not be nil.
func initBlockedServices(ctx context.Context, l *slog.Logger) {
	svcLen := len(blockedServices)
	serviceIDs = make([]string, svcLen)
	serviceRules = make(map[string][]*rules.NetworkRule, svcLen)

	for i, s := range blockedServices {
		netRules := make([]*rules.NetworkRule, 0, len(s.Rules))
		for _, text := range s.Rules {
			rule, err := rules.NewNetworkRule(text, rulelist.IDBlockedService)
			if err == nil {
				netRules = append(netRules, rule)

				continue
			}

			l.ErrorContext(
				ctx,
				"parsing blocked service rule",
				"svc", s.ID,
				"rule", text,
				slogutil.KeyError, err,
			)
		}

		serviceIDs[i] = s.ID
		serviceRules[s.ID] = netRules
	}

	slices.Sort(serviceIDs)

	l.DebugContext(ctx, "initialized services", "svc_len", svcLen)
}

// ServiceSchedule represents a schedule for a single blocked service.
type ServiceSchedule struct {
	ID       string           `json:"id" yaml:"id"`
	Schedule *schedule.Weekly `json:"schedule" yaml:"schedule"`
}

// BlockedServices is the configuration of blocked services.
//
// TODO(s.chzhen):  Move to a higher-level package to allow importing the client
// package into the filtering package.
type BlockedServices struct {
	// Schedule is blocked services schedule for every day of the week.
	// Deprecated: Use Services for per-service scheduling.
	Schedule *schedule.Weekly `json:"schedule" yaml:"schedule"`

	// IDs is the names of blocked services.
	IDs []string `json:"ids" yaml:"ids"`

	// Services is the list of services with their individual schedules.
	// When a service has a schedule, it is blocked during the scheduled times
	// and not blocked outside those times (regardless of whether it's in IDs).
	Services []ServiceSchedule `json:"services" yaml:"services"`
}

// Clone returns a deep copy of blocked services.
func (s *BlockedServices) Clone() (c *BlockedServices) {
	if s == nil {
		return nil
	}

	services := make([]ServiceSchedule, len(s.Services))
	for i, svc := range s.Services {
		services[i] = ServiceSchedule{
			ID:       svc.ID,
			Schedule: svc.Schedule.Clone(),
		}
	}

	return &BlockedServices{
		Schedule: s.Schedule.Clone(),
		IDs:      slices.Clone(s.IDs),
		Services: services,
	}
}

// FilterUnknownIDs filters out unknown service IDs within s and logs them at
// warning level.  It does nothing if s is nil.
func (s *BlockedServices) FilterUnknownIDs(ctx context.Context, logger *slog.Logger) {
	if s == nil {
		// [BlockedServices.Validate] handles this case.
		return
	}

	s.IDs = slices.DeleteFunc(s.IDs, func(id string) (ok bool) {
		_, isKnown := serviceRules[id]
		if !isKnown {
			logger.WarnContext(ctx, "filtered unknown service", "id", id)
		}

		return !isKnown
	})
}

// type check
var _ validate.Interface = (*BlockedServices)(nil)

// Validate implements the [validate.Interface] interface for *BlockedServices.
func (s *BlockedServices) Validate() (err error) {
	if s == nil {
		return errors.ErrNoValue
	}

	var errs []error
	for _, id := range s.IDs {
		_, ok := serviceRules[id]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown blocked-service %q", id))
		}
	}

	for _, svc := range s.Services {
		_, ok := serviceRules[svc.ID]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown blocked-service %q", svc.ID))
		}
	}

	return errors.Join(errs...)
}

// ApplyBlockedServices - set blocked services settings for this DNS request
func (d *DNSFilter) ApplyBlockedServices(setts *Settings) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	setts.ServicesRules = []ServiceEntry{}

	bsvc := d.conf.BlockedServices
	now := time.Now()

	// Start with default blocked services
	blockedIDs := slices.Clone(bsvc.IDs)

	// Apply per-service schedules
	for _, svc := range bsvc.Services {
		isScheduled := svc.Schedule != nil && svc.Schedule.Contains(now)
		isInDefaultList := slices.Contains(blockedIDs, svc.ID)

		if isScheduled && !isInDefaultList {
			// Service has active schedule but not in default list - add it
			blockedIDs = append(blockedIDs, svc.ID)
		} else if !isScheduled && isInDefaultList {
			// Service has no active schedule but is in default list - remove it
			blockedIDs = slices.DeleteFunc(blockedIDs, func(id string) bool {
				return id == svc.ID
			})
		}
	}

	// Apply legacy single schedule if no per-service schedules exist
	if len(bsvc.Services) == 0 && bsvc.Schedule != nil {
		if bsvc.Schedule.Contains(now) {
			d.ApplyBlockedServicesList(setts, blockedIDs)
		}

		return
	}

	// Apply the computed blocked services list
	d.ApplyBlockedServicesList(setts, blockedIDs)
}

// ApplyBlockedServicesList appends filtering rules to the settings.
func (d *DNSFilter) ApplyBlockedServicesList(setts *Settings, list []string) {
	for _, name := range list {
		rules, ok := serviceRules[name]
		if !ok {
			d.logger.ErrorContext(context.TODO(), "unknown service name", "name", name)

			continue
		}

		setts.ServicesRules = append(setts.ServicesRules, ServiceEntry{
			Name:  name,
			Rules: rules,
		})
	}
}

func (d *DNSFilter) handleBlockedServicesIDs(w http.ResponseWriter, r *http.Request) {
	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, serviceIDs)
}

func (d *DNSFilter) handleBlockedServicesAll(w http.ResponseWriter, r *http.Request) {
	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, struct {
		BlockedServices []blockedService `json:"blocked_services"`
		ServiceGroups   []serviceGroup   `json:"groups"`
	}{
		BlockedServices: blockedServices,
		ServiceGroups:   serviceGroups,
	})
}

// handleBlockedServicesList is the handler for the GET
// /control/blocked_services/list HTTP API.
//
// Deprecated:  Use handleBlockedServicesGet.
func (d *DNSFilter) handleBlockedServicesList(w http.ResponseWriter, r *http.Request) {
	var list []string
	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		list = d.conf.BlockedServices.IDs
	}()

	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, list)
}

// handleBlockedServicesSet is the handler for the POST
// /control/blocked_services/set HTTP API.
//
// Deprecated:  Use handleBlockedServicesUpdate.
func (d *DNSFilter) handleBlockedServicesSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	list := []string{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, d.logger, r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		d.conf.BlockedServices.IDs = list
		d.logger.DebugContext(ctx, "updated blocked services list", "len", len(list))
	}()

	d.conf.ConfModifier.Apply(ctx)
}

// handleBlockedServicesGet is the handler for the GET
// /control/blocked_services/get HTTP API.
func (d *DNSFilter) handleBlockedServicesGet(w http.ResponseWriter, r *http.Request) {
	var bsvc *BlockedServices
	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()

		bsvc = d.conf.BlockedServices.Clone()
	}()

	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, bsvc)
}

// handleBlockedServicesUpdate is the handler for the PUT
// /control/blocked_services/update HTTP API.
func (d *DNSFilter) handleBlockedServicesUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := d.logger

	bsvc := &BlockedServices{}
	err := json.NewDecoder(r.Body).Decode(bsvc)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	err = bsvc.Validate()
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusUnprocessableEntity, "validating: %s", err)

		return
	}

	if bsvc.Schedule == nil {
		bsvc.Schedule = schedule.EmptyWeekly()
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		d.conf.BlockedServices = bsvc
	}()

	l.DebugContext(ctx, "updated blocked services schedule", "len", len(bsvc.IDs))

	d.conf.ConfModifier.Apply(ctx)
}
