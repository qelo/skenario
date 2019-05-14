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
	"context"
	"github.com/knative/serving/pkg/autoscaler"
	"time"

	"skenario/pkg/simulator"
)

type FakeEnvironment struct {
	Movements   []simulator.Movement
	TheTime     time.Time
	TheHaltTime time.Time
}

func (fe *FakeEnvironment) AddToSchedule(movement simulator.Movement) (added bool) {
	fe.Movements = append(fe.Movements, movement)
	return true
}

func (fe *FakeEnvironment) Run() (completed []simulator.CompletedMovement, ignored []simulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func (fe *FakeEnvironment) CurrentMovementTime() time.Time {
	return fe.TheTime
}

func (fe *FakeEnvironment) HaltTime() time.Time {
	return fe.TheHaltTime
}

func (fe *FakeEnvironment) Context() context.Context {
	return context.Background()
}

type FakeReplica struct {
	ActivateCalled           bool
	DeactivateCalled         bool
	RequestsProcessingCalled bool
	StatCalled               bool
	FakeReplicaNum           int
	ProcessingStock          RequestsProcessingStock
}

func (*FakeReplica) Name() simulator.EntityName {
	return "Replica"
}

func (*FakeReplica) Kind() simulator.EntityKind {
	return "Replica"
}

func (fr *FakeReplica) Activate() {
	fr.ActivateCalled = true
}

func (fr *FakeReplica) Deactivate() {
	fr.DeactivateCalled = true
}

func (fr *FakeReplica) RequestsProcessing() RequestsProcessingStock {
	fr.RequestsProcessingCalled = true
	if fr.ProcessingStock == nil {
		return NewRequestsProcessingStock(new(FakeEnvironment), fr.FakeReplicaNum, simulator.NewSinkStock("fake-sink", "Request"), 100)
	} else {
		return fr.ProcessingStock
	}
}

func (fr *FakeReplica) Stat() autoscaler.Stat {
	fr.StatCalled = true
	return autoscaler.Stat{}
}
