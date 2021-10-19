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

package v1

func (t *Trench) Equals(t2 *Trench) bool {
	if t == nil || t2 == nil {
		return false
	}
	return t.GetName() == t2.GetName()
}

func (c *Conduit) Equals(c2 *Conduit) bool {
	if c == nil || c2 == nil {
		return false
	}
	names := true
	if c.GetName() != "" && c2.GetName() != "" {
		names = c.GetName() == c2.GetName()
	}
	return names && c.GetTrench().Equals(c2.GetTrench())
}

func (s *Stream) Equals(s2 *Stream) bool {
	if s == nil || s2 == nil {
		return false
	}
	names := true
	if s.GetName() != "" && s2.GetName() != "" {
		names = s.GetName() == s2.GetName()
	}
	return names && s.GetConduit().Equals(s2.GetConduit())
}

func (f *Flow) Equals(f2 *Flow) bool {
	if f == nil || f2 == nil {
		return false
	}
	names := true
	if f.GetName() != "" && f2.GetName() != "" {
		names = f.GetName() == f2.GetName()
	}
	return names && f.GetStream().Equals(f2.GetStream())
}

func (v *Vip) Equals(v2 *Vip) bool {
	if v == nil || v2 == nil {
		return false
	}
	names := true
	if v.GetName() != "" && v2.GetName() != "" {
		names = v.GetName() == v2.GetName()
	}
	return names && v.GetTrench().Equals(v2.GetTrench())
}

func (a *Attractor) Equals(a2 *Attractor) bool {
	if a == nil || a2 == nil {
		return false
	}
	names := true
	if a.GetName() != "" && a2.GetName() != "" {
		names = a.GetName() == a2.GetName()
	}
	return names && a.GetTrench().Equals(a2.GetTrench())
}

func (g *Gateway) Equals(g2 *Gateway) bool {
	if g == nil || g2 == nil {
		return false
	}
	names := true
	if g.GetName() != "" && g2.GetName() != "" {
		names = g.GetName() == g2.GetName()
	}
	return names && g.GetTrench().Equals(g2.GetTrench())
}

func (t *Target) Equals(t2 *Target) bool {
	if t == nil || t2 == nil {
		return false
	}
	status := t.GetStatus() == t2.GetStatus()
	if t.GetStatus() == Target_ANY || t2.GetStatus() == Target_ANY {
		status = true
	}
	stream := t.GetStream().Equals(t2.GetStream())
	if t.GetStream() == nil && t2.GetStream() == nil {
		stream = true
	}
	return status && t.GetType() == t2.GetType() && stream
}
