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

type AutoscalerTicktockStock interface {
	simulator.ThroughStock
}

type autoscalerTicktockStock struct {
	env              simulator.Environment
	cluster          ClusterModel
	autoscalerEntity simulator.Entity
	desiredSource    simulator.ThroughStock
	desiredSink      simulator.ThroughStock
}

func (asts *autoscalerTicktockStock) Name() simulator.StockName {
	return "Autoscaler Ticktock"
}

func (asts *autoscalerTicktockStock) KindStocked() simulator.EntityKind {
	return "HPAAutoscaler"
}

func (asts *autoscalerTicktockStock) Count() uint64 {
	return 1
}

func (asts *autoscalerTicktockStock) EntitiesInStock() []*simulator.Entity {
	return []*simulator.Entity{&asts.autoscalerEntity}
}

func (asts *autoscalerTicktockStock) Remove() simulator.Entity {
	return asts.autoscalerEntity
}

func (asts *autoscalerTicktockStock) Add(entity simulator.Entity) error {
	if asts.autoscalerEntity != entity {
		return fmt.Errorf("'%+v' is different from the entity given at creation time, '%+v'", entity, asts.autoscalerEntity)
	}

	currentTime := asts.env.CurrentMovementTime()

	asts.cluster.RecordToAutoscaler(&currentTime)
	autoscalerDesired, err := asts.env.Plugin().Scale(currentTime.UnixNano())
	if err != nil {
		panic(err)
	}

	delta := autoscalerDesired - int32(asts.cluster.Desired().Count())

	if delta > 0 {
		for i := int32(0); i < delta; i++ {
			err := asts.desiredSource.Add(simulator.NewEntity("Desired", "Desired"))
			if err != nil {
				return err
			}

			asts.env.AddToSchedule(simulator.NewMovement(
				"increase_desired",
				currentTime.Add(1*time.Nanosecond),
				asts.desiredSource,
				asts.cluster.Desired(),
			))
		}
	} else if delta < 0 {
		for i := delta; i < 0; i++ {
			asts.env.AddToSchedule(simulator.NewMovement(
				"reduce_desired",
				currentTime.Add(1*time.Nanosecond),
				asts.cluster.Desired(),
				asts.desiredSink,
			))
		}
	} else {
		// do nothing
	}

	//calculate CPU utilization
	asts.calculateCPUUtilization()

	return nil
}

func (asts *autoscalerTicktockStock) calculateCPUUtilization() {
	countActiveReplicas := 0.0
	totalCPUUtilization := 0.0 // total cpuUtilization for all active replicas in percentage

	for _, en := range asts.cluster.ActiveStock().EntitiesInStock() {
		replica := (*en).(*replicaEntity)
		totalCPUUtilization += replica.occupiedCPUCapacityMillisPerSecond * 100 / replica.totalCPUCapacityMillisPerSecond
		countActiveReplicas++
	}
	if countActiveReplicas > 0 {
		averageCPUUtilizationPerReplica := simulator.CPUUtilization{CPUUtilization: totalCPUUtilization / countActiveReplicas,
			CalculatedAt: asts.env.CurrentMovementTime()}
		asts.env.AppendCPUUtilization(&averageCPUUtilizationPerReplica)
	}
}

func NewAutoscalerTicktockStock(env simulator.Environment, scalerEntity simulator.Entity, cluster ClusterModel) AutoscalerTicktockStock {
	return &autoscalerTicktockStock{
		env:              env,
		cluster:          cluster,
		autoscalerEntity: scalerEntity,
		desiredSource:    simulator.NewThroughStock("DesiredSource", "Desired"),
		desiredSink:      simulator.NewThroughStock("DesiredSink", "Desired"),
	}
}
