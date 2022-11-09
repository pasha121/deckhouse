/*
Copyright 2021 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"d8.io/upmeter/pkg/check"
	dbcontext "d8.io/upmeter/pkg/db/context"
	"d8.io/upmeter/pkg/db/dao"
	"d8.io/upmeter/pkg/monitor/downtime"
	"d8.io/upmeter/pkg/registry"
	"d8.io/upmeter/pkg/server/entity"
	"d8.io/upmeter/pkg/server/ranges"
)

type PublicStatusResponse struct {
	Status PublicStatus  `json:"status"`
	Rows   []GroupStatus `json:"rows"`
}

type GroupStatus struct {
	Group  string              `json:"group"`
	Status PublicStatus        `json:"status"`
	Probes []ProbeAvailability `json:"probes"`
}

type ProbeAvailability struct {
	// Probe is the name of the probe
	Probe string `json:"probe"`

	// Availability is the ratio represented as a fraction of 1, i.e. it is from 0 to 1 and it must
	// neve be negative
	Availability float64 `json:"availability"`
}

type PublicStatus string

const (
	StatusOperational PublicStatus = "Operational"
	StatusDegraded    PublicStatus = "Degraded"
	StatusOutage      PublicStatus = "Outage"
)

type PublicStatusHandler struct {
	DbCtx           *dbcontext.DbContext
	DowntimeMonitor *downtime.Monitor
	ProbeLister     registry.ProbeLister
}

func (h *PublicStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infoln("PublicStatus", r.RemoteAddr, r.RequestURI)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		message := fmt.Sprintf("%s not allowed, use GET\n", r.Method)
		fmt.Fprint(w, jsonError(message))
		return
	}

	statuses, status, err := h.getStatusSummary()
	if err != nil {
		log.Errorf("Cannot get status summary: %v", err)
		// Skipping the error because the JSON structure is defined in advance.
		out, _ := json.Marshal(&PublicStatusResponse{
			Rows:   []GroupStatus{},
			Status: "No data for last 15 min",
		})
		w.WriteHeader(http.StatusOK)
		w.Write(out)
		return
	}

	// Skipping the error because the JSON structure is defined in advance.
	out, _ := json.Marshal(&PublicStatusResponse{
		Rows:   statuses,
		Status: status,
	})
	w.Write(out)
}

// getStatusSummary returns total statuses for each group for the current partial 5m timeslot plus
// previous full 5m timeslot.
func (h *PublicStatusHandler) getStatusSummary() ([]GroupStatus, PublicStatus, error) {
	daoCtx := h.DbCtx.Start()
	defer daoCtx.Stop()

	rng := threeStepRange(time.Now())
	log.Infof("Request public status from=%d to=%d at %d", rng.From, rng.To, time.Now().Unix())

	muteTypes := []string{
		"Maintenance",
		"InfrastructureMaintenance",
		"InfrastructureAccident",
	}

	groups := h.ProbeLister.Groups()
	groupStatuses := make([]GroupStatus, 0, len(groups))
	for _, group := range groups {
		// The group overall status
		groupRef := check.ProbeRef{
			Group: group,
			Probe: dao.GroupAggregation,
		}

		filter := &statusFilter{
			stepRange:         rng,
			probeRef:          groupRef,
			muteDowntimeTypes: muteTypes,
		}

		resp, err := getStatus(h.DbCtx, h.DowntimeMonitor, filter)
		if err != nil {
			log.Errorf("cannot calculate status for group %s: %v", group, err)
			return nil, StatusOutage, err
		}

		summary, err := pickSummary(groupRef, resp.Statuses)
		if err != nil {
			log.Errorf("generating summary %s: %v", group, err)
			return nil, StatusOutage, err
		}

		probeAvails := make([]ProbeAvailability, 0, len(resp.Statuses))
		// Uptime per probe
		for _, probeRef := range h.ProbeLister.Probes() {
			if probeRef.Group != group {
				// not so many probes in the list, so it is ok to iterate
				continue
			}

			filter := &statusFilter{
				stepRange:         rng,
				probeRef:          probeRef,
				muteDowntimeTypes: muteTypes,
			}
			resp, err := getStatus(h.DbCtx, h.DowntimeMonitor, filter)
			if err != nil {
				log.Errorf("cannot calculate status for group %s: %v", group, err)
				return nil, StatusOutage, err
			}

			summary, err := pickSummary(probeRef, resp.Statuses)
			if err != nil {
				log.Errorf("generating summary %s: %v", group, err)
				return nil, StatusOutage, err
			}

			av := calculateAvailability(summary)
			if av < 0 {
				continue
			}
			probeAvails = append(probeAvails, ProbeAvailability{
				Probe:        probeRef.Probe,
				Availability: av,
			})
		}

		gs := GroupStatus{
			Group:  group,
			Status: calculateStatus(summary),
			Probes: probeAvails,
		}
		groupStatuses = append(groupStatuses, gs)
	}

	totalStatus := calculateTotalStatus(groupStatuses)
	return groupStatuses, totalStatus, nil
}

// Negative availability means we have no valid data
func calculateAvailability(summary []entity.EpisodeSummary) float64 {
	var uptime, total float64
	for _, s := range summary {
		uptime += float64(s.Up + s.Unknown)
		total += float64(s.Up + s.Unknown + s.Down)
	}
	if total == 0 {
		return -1
	}
	return uptime / total
}

func threeStepRange(now time.Time) ranges.StepRange {
	step := 5 * time.Minute
	slotStart := now.Truncate(step)
	from := slotStart.Add(-2 * step)
	to := slotStart.Add(step)
	return ranges.NewStepRange(from.Unix(), to.Unix(), int64(step.Seconds()))
}

var ErrNoData = fmt.Errorf("no data")

// pickSummary makes assertions
func pickSummary(ref check.ProbeRef, statuses map[string]map[string][]entity.EpisodeSummary) ([]entity.EpisodeSummary, error) {
	g := ref.Group
	p := ref.Probe

	if _, ok := statuses[g]; !ok {
		return nil, fmt.Errorf("%w for group '%s'", ErrNoData, g)
	}
	if _, ok := statuses[g][p]; !ok {
		return nil, fmt.Errorf("%s for probe '%s/%s'", ErrNoData, g, p)
	}

	episodeSummaries := statuses[g][p]
	n := len(episodeSummaries) - 1 // ignore summary column in the end
	if n != 3 {
		return nil, fmt.Errorf("unexpected count %d!=3 for probe '%s/%s'", n, g, p)
	}

	return episodeSummaries[:3], nil
}

// calculateStatus returns the status for a group.
//
// Input array should have 3 elements
func calculateStatus(sums []entity.EpisodeSummary) PublicStatus {
	slotSize := 5 * time.Minute

	var prev, cur entity.EpisodeSummary
	if len(sums) == 2 || sums[2].NoData == slotSize {
		// we actually have only two slots of data at most
		prev, cur = sums[0], sums[1]
	} else {
		// ignore 1st slot
		prev, cur = sums[1], sums[2]
	}

	// Ignore empty EpisodeSummary, i.e. when NoData equals slot size
	if cur.Down == 0 && prev.Down == 0 &&
		(cur.Up > 0 || (prev.Up > 0 && cur.Up == 0 && cur.NoData == slotSize)) {
		return StatusOperational
	}

	if cur.Up == 0 && cur.Muted == 0 &&
		prev.Up == 0 && prev.Muted == 0 &&
		(cur.Down > 0 || (prev.Down > 0 && cur.Down == 0 && cur.NoData == slotSize)) {
		return StatusOutage
	}

	return StatusDegraded
}

// calculateTotalStatus returns total cluster status.
func calculateTotalStatus(statuses []GroupStatus) PublicStatus {
	warn := false
	for _, info := range statuses {
		switch info.Status {
		case StatusDegraded:
			warn = true
		case StatusOutage:
			return StatusOutage
		}
	}
	if warn {
		return StatusDegraded
	}
	return StatusOperational
}

func jsonError(msg string) string {
	return fmt.Sprintf(`{"error": %q}`, msg)
}
