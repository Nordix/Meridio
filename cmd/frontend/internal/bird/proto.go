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

package bird

import (
	"bufio"
	"regexp"
	"strings"
)

var regexProto *regexp.Regexp = regexp.MustCompile(`(NBR-\S+)\s+(Static|BGP)\s+\S+\s+(\S+)\s+\S+(.*)`)
var regexBirdcTitle *regexp.Regexp = regexp.MustCompile(`BIRD|Name\s+Proto`)

// ParseProtocols -
// Parses output of 'birdc show protocols all' to pass details of a protocol
// session to the provided func pointer (while also writing relevant information
// to out log string)
func ParseProtocols(input string, out *string, f func(name string, options ...Option)) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	//logrus.Infof("ParseProtocols: \ninput: %v", input)

	for scanner.Scan() {
		if match := regexProto.FindStringSubmatch(scanner.Text()); match != nil {
			if out != nil {
				*out += scanner.Text()
			}
			// get name and other attributes of the particular BIRD protocol session
			name := match[1]
			//logrus.Infof("ParseProtocols: name: %v", name)
			opts := []Option{
				WithName(name),
				WithProto(match[2]),
				WithState(match[3]),
				WithInfo(match[4]),
			}
			if out != nil {
				opts = append(opts, WithOutLog(out))
			}
			f(name, opts...)
		} else if strings.Contains(scanner.Text(), `Neighbor address`) {
			if out != nil {
				*out += scanner.Text() + "\n"
			}
		} else if ok := regexBirdcTitle.MatchString(scanner.Text()); ok {
			if out != nil {
				*out += scanner.Text() + "\n"
			}
		}
	}
}

// ProtocolDown -
// Determines if protocol is down
func ProtocolDown(p *Protocol) bool {
	rc := false
	switch p.Proto() {
	case BGP:
		rc = stateDown(p) || !strings.Contains(p.Info(), "Established")
	case STATIC:
		{
			rc = stateDown(p) || bfdDown(p)
			p.Log("\n")
		}
	}

	return rc
}

// stateDown -
// Checks if protocol state is down
func stateDown(p *Protocol) bool {
	return p.State() != "up"
}

// bfdDown -
// Checks associated bfd session's state if any
func bfdDown(p *Protocol) bool {
	regexBFD := regexp.MustCompile(p.Neighbor() + `\s+` + p.Interface() + `\s+(\S+).*`)
	scanner := bufio.NewScanner(strings.NewReader(p.BfdSessions()))
	for scanner.Scan() {
		if bfdMatch := regexBFD.FindStringSubmatch(scanner.Text()); bfdMatch != nil {
			p.Log(" bfd: " + bfdMatch[0])
			return bfdMatch[1] != "Up"
		}
	}
	return false
}

//-------------------------------------------------------------------------------

func NewProtocolMap() protocolMap {
	return map[int]string{}
}

type protocolMap map[int]string

func NewProtocol(options ...Option) *Protocol {
	opts := &protoOptions{
		m: NewProtocolMap(),
	}
	for _, opt := range options {
		opt(opts)
	}

	return &Protocol{
		protoMap: opts.m,
		log:      opts.log,
	}
}

type Protocol struct {
	protoMap protocolMap
	log      *string
}

func (p *Protocol) Name() string {
	return p.protoMap[protoName]
}

func (p *Protocol) Proto() string {
	return p.protoMap[protoProto]
}

func (p *Protocol) State() string {
	return p.protoMap[protoState]
}

func (p *Protocol) Info() string {
	return p.protoMap[protoInfo]
}

func (p *Protocol) Interface() string {
	return p.protoMap[protoItf]
}

func (p *Protocol) Neighbor() string {
	return p.protoMap[protoNbr]
}

func (p *Protocol) BfdSessions() string {
	return p.protoMap[bfdSessions]
}

func (p *Protocol) Log(out string) {
	if p.log != nil {
		*p.log += out
	}
}
