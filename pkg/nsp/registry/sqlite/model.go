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

// Equals reports whether a stored target matches an incoming one. Set uses it
// to skip refreshes that change nothing.
//
// It compares the columns of the targets table: ID, Ips, Context, Status and
// Type. ID is built from the IPs, type, stream, conduit and trench names, so
// equal IDs mean the same stream, conduit and trench names were requested.
//
// Two fields are not compared:
//   - StreamName: it is only filled in when the target is written to the DB,
//     so it is always empty on the incoming target and would never match.
//   - Stream: the stream's conduit and trench are stored in the streams table,
//     not in the targets table, and Set does not load them.
//
// Limitation: the streams table has one row per stream name, shared by all
// targets of that stream. Another target's write can change that row's conduit
// (see clearConflicts). Equals does not notice, because it does not read the
// streams row. If the streams, conduits or trenches tables gain a column that is
// not part of the ID, Equals must compare it.
//
// Ips is compared as stored, without sorting. A reordered IP list gives a
// different ID, so it never reaches this comparison.
func (t *Target) Equals(t2 *Target) bool {
	if t == nil || t2 == nil {
		return t == t2
	}
	return t.ID == t2.ID &&
		t.Ips == t2.Ips &&
		t.Context == t2.Context &&
		t.Status == t2.Status &&
		t.Type == t2.Type
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
