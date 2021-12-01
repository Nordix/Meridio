/*
Copyright (c) 2021 Nordix Foundation

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

package sqlite

import (
	"fmt"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type Target struct {
	ID         string `gorm:"primaryKey"`
	Ips        string
	Context    string
	Status     int32
	Type       int32
	StreamName string
	Stream     Stream `gorm:"foreignKey:StreamName"`
}

type Stream struct {
	Name        string `gorm:"primaryKey"`
	ConduitName string
	Conduit     Conduit `gorm:"foreignKey:ConduitName"`
}

type Conduit struct {
	Name       string `gorm:"primaryKey"`
	TrenchName string
	Trench     Trench `gorm:"foreignKey:TrenchName"`
}

type Trench struct {
	Name string `gorm:"primaryKey"`
}

func NSPTargetToSQLTarget(target *nspAPI.Target) *Target {
	if target == nil {
		return nil
	}
	return &Target{
		ID:      GetTargetID(target),
		Ips:     SerializeIPs(target.GetIps()),
		Context: SerializeContext(target.GetContext()),
		Status:  SerializeStatus(target.GetStatus()),
		Type:    SerializeType(target.GetType()),
		Stream:  NSPTStreamToSQLStream(target.GetStream()),
	}
}

func NSPTStreamToSQLStream(stream *nspAPI.Stream) Stream {
	if stream == nil {
		return Stream{}
	}
	return Stream{
		Name:    stream.GetName(),
		Conduit: NSPTConduitToSQLConduit(stream.GetConduit()),
	}
}

func NSPTConduitToSQLConduit(conduit *nspAPI.Conduit) Conduit {
	if conduit == nil {
		return Conduit{}
	}
	return Conduit{
		Name:   conduit.GetName(),
		Trench: NSPTTrenchToSQLTrench(conduit.GetTrench()),
	}
}

func NSPTTrenchToSQLTrench(trench *nspAPI.Trench) Trench {
	if trench == nil {
		return Trench{}
	}
	return Trench{
		Name: trench.GetName(),
	}
}

func SQLTargetToNSPTarget(target *Target) *nspAPI.Target {
	if target == nil {
		return nil
	}
	return &nspAPI.Target{
		Ips:     DeserializeIPs(target.Ips),
		Context: DeserializeContext(target.Context),
		Status:  DeserializeStatus(target.Status),
		Type:    DeserializeType(target.Type),
		Stream:  SQLTStreamToNSPStream(&target.Stream),
	}
}

func SQLTStreamToNSPStream(stream *Stream) *nspAPI.Stream {
	if stream == nil {
		return nil
	}
	return &nspAPI.Stream{
		Name:    stream.Name,
		Conduit: SQLTConduitToNSPConduit(&stream.Conduit),
	}
}

func SQLTConduitToNSPConduit(conduit *Conduit) *nspAPI.Conduit {
	if conduit == nil {
		return nil
	}
	return &nspAPI.Conduit{
		Name:   conduit.Name,
		Trench: SQLTTrenchToNSPTrench(&conduit.Trench),
	}
}

func SQLTTrenchToNSPTrench(trench *Trench) *nspAPI.Trench {
	if trench == nil {
		return nil
	}
	return &nspAPI.Trench{
		Name: trench.Name,
	}
}

func GetTargetID(target *nspAPI.Target) string {
	if target == nil {
		return ""
	}
	return fmt.Sprintf("%s-%d-%s.%s.%s",
		SerializeIPs(target.GetIps()),
		SerializeType(target.GetType()),
		getTargetStream(target),
		getTargetConduit(target),
		getTargetTrench(target))
}

func getTargetStream(target *nspAPI.Target) string {
	if target.GetStream() == nil {
		return ""
	}
	return target.GetStream().GetName()
}

func getTargetConduit(target *nspAPI.Target) string {
	if target.GetStream() == nil || target.GetStream().GetConduit() == nil {
		return ""
	}
	return target.GetStream().GetConduit().GetName()
}

func getTargetTrench(target *nspAPI.Target) string {
	if target.GetStream() == nil || target.GetStream().GetConduit() == nil || target.GetStream().GetConduit().GetTrench() == nil {
		return ""
	}
	return target.GetStream().GetConduit().GetTrench().GetName()
}
