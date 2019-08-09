/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package model

import (
	"fmt"
	"time"

	"skenario/pkg/simulator"
)

type RequestsProcessingStock interface {
	simulator.ThroughStock
	RequestCount() int32
}

type cpuUsage struct {
	window           time.Duration
	activeTimeSlices [][2]time.Time
}

func NewCpuUsage(window time.Duration) *cpuUsage {
	return &cpuUsage{
		window:           window,
		activeTimeSlices: make([][2]time.Time, 0),
	}
}

func (c *cpuUsage) active(from, to time.Time) {
	c.activeTimeSlices = append(c.activeTimeSlices, [2]time.Time{from, to})
}

func (c *cpuUsage) trim(now time.Time) {
	trimmed := make([][2]time.Time, 0)
	for _, t := range c.activeTimeSlices {
		if t[1].Before(now.Add(-c.window)) {
			continue
		}
		if t[0].Before(now.Add(-c.window)) {
			t[0] = now.Add(-c.window)
		}
		trimmed = append(trimmed, t)
	}
	c.activeTimeSlices = trimmed
}

func (c *cpuUsage) usage(now time.Time) float64 {
	c.trim(now)
	var activeNanos int64
	for _, a := range c.activeTimeSlices {
		activeNanos += a[1].Sub(a[0]).Nanoseconds()
	}
	return float64(activeNanos) / float64(c.window.Nanoseconds())
}

type requestsProcessingStock struct {
	env           simulator.Environment
	replicaNumber int

	// Internal process accounting.
	processesActive     simulator.ThroughStock
	processesOnCpu      simulator.ThroughStock
	processesTerminated simulator.ThroughStock
	cpuUsage            *cpuUsage

	requestsComplete      simulator.SinkStock
	numRequestsSinceLast  int32
	replicaMaxRPSCapacity int64 // unused
}

func (rps *requestsProcessingStock) Name() simulator.StockName {
	name := fmt.Sprintf("%s [%d]", rps.processesActive.Name(), rps.replicaNumber)
	return simulator.StockName(name)
}

func (rps *requestsProcessingStock) KindStocked() simulator.EntityKind {
	return rps.processesActive.KindStocked()
}

func (rps *requestsProcessingStock) Count() uint64 {
	return rps.processesActive.Count() + rps.processesOnCpu.Count()
}

func (rps *requestsProcessingStock) EntitiesInStock() []*simulator.Entity {
	entities := make([]*simulator.Entity, rps.processesActive.Count()+rps.processesOnCpu.Count())
	entities = append(entities, rps.processesActive.EntitiesInStock()...)
	entities = append(entities, rps.processesOnCpu.EntitiesInStock()...)
	return entities
}

func (rps *requestsProcessingStock) Remove() simulator.Entity {
	if rps.processesTerminated.Count() > 0 {
		return rps.processesTerminated.Remove()
	}
	return nil
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	// TODO: this isn't correct anymore because it's used for interrupts.
	//rps.numRequestsSinceLast++

	req, ok := entity.(*requestEntity)
	if !ok {
		return fmt.Errorf("requests processing stock only supports request entities. got %T", entity)
	}
	now := rps.env.CurrentMovementTime()
	if req.startTime == nil {
		req.startTime = &now
	}

	// Enqueue or complete the request.
	if req.cpuSecondsRemaining() > 0 {
		err := rps.processesActive.Add(entity)
		if err != nil {
			return err
		}
	} else {
		err := rps.processesTerminated.Add(entity)
		if err != nil {
			return err
		}
		rps.env.AddToSchedule(simulator.NewMovement(
			"complete_request",
			now.Add(time.Nanosecond),
			rps,
			rps.requestsComplete,
		))
		// latency := now.Sub(*req.startTime)
		// log.Printf("latecy: %v\n", latency)
	}

	// Fill the CPU and schedule an interrupt.
	if rps.processesOnCpu.Count() == 0 && rps.processesActive.Count() > 0 {
		req := rps.processesActive.Remove().(*requestEntity)
		interruptAfter := req.cpuSecondsRemaining()
		if interruptAfter > 200*time.Millisecond {
			interruptAfter = 200 * time.Millisecond
		}
		interruptAt := now.Add(interruptAfter)
		req.cpuSecondsConsumed += interruptAfter
		rps.env.AddToSchedule(simulator.NewMovement(
			"interrupt_process",
			interruptAt,
			rps.processesOnCpu,
			rps,
		))
		rps.cpuUsage.active(now, interruptAt)
		return rps.processesOnCpu.Add(entity)
	}
	return nil
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestSink simulator.SinkStock, replicaMaxRPSCapacity int64) RequestsProcessingStock {
	return &requestsProcessingStock{
		env:                   env,
		processesActive:       simulator.NewThroughStock("RequestsProcessing", "Request"),
		processesOnCpu:        simulator.NewThroughStock("RequestsProcessing", "Request"),
		processesTerminated:   simulator.NewThroughStock("RequestsProcessing", "Request"),
		replicaNumber:         replicaNumber,
		requestsComplete:      requestSink,
		replicaMaxRPSCapacity: replicaMaxRPSCapacity,
		cpuUsage:              NewCpuUsage(15 * time.Second),
	}
}
