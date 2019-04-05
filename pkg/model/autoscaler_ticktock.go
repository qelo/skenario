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

	"knative-simulator/pkg/simulator"
)

type AutoscalerTicktockStock interface {
	simulator.ThroughStock
}

type autoscalerTicktockStock struct {
	autoscalerEntity simulator.Entity
}

func (asts *autoscalerTicktockStock) Name() simulator.StockName {
	return "Autoscaler Ticktock"
}

func (asts *autoscalerTicktockStock) KindStocked() simulator.EntityKind {
	return "KnativeAutoscaler"
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
	return nil
}

func NewAutoscalerTicktockStock(scalerEntity simulator.Entity) AutoscalerTicktockStock {
	return &autoscalerTicktockStock{
		autoscalerEntity: scalerEntity,
	}
}