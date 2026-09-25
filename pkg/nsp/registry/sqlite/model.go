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

// Equals reports whether two target rows are identical in their persisted
// payload. Unlike nspAPI.Target.Equals (a loose selector match that wildcards
// status and ignores Ips/Context), this is a strict comparison used to detect
// whether a Set() actually changes the stored row.
//
// The comparison is intentionally defensive: it compares all persisted scalar
// columns of Target, including Ips and Type which are already components of the
// primary key (see GetTargetID). Comparing them is redundant while the ID
// matches, but keeps the check correct if the ID definition ever changes.
//
// StreamName is deliberately excluded. NSPTargetToSQLTarget populates only the
// nested Stream association, not the top-level StreamName column (GORM derives
// that column from the association on write), so a freshly converted incoming
// model always has StreamName == "" while a row loaded from the DB has it set.
// Including it would make Equals always report inequality and defeat the fast
// path.
//
// The nested Stream association is not compared either, and this is safe only
// because every field it carries — Stream.Name, Conduit.Name, Trench.Name — is a
// component of the ID. An equal ID therefore already implies an identical
// stream/conduit/trench chain, so no persisted change there can be missed.
// CAUTION: if Stream, Conduit or Trench ever gain a persisted attribute that is
// NOT part of the ID (e.g. a stored setting on the stream), that attribute could
// change under a stable ID and would be silently skipped here — Equals (and the
// association write path) would then need to account for it.
//
// Ips is an order-sensitive join of the address list (SerializeIPs does not
// sort). Because Ips is part of the primary key, a reordered address list yields
// a different ID and is handled as a distinct row upstream (Create path) rather
// than reaching an ID-matched equality check here, so the ordering does not
// cause false inequality in practice.
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
