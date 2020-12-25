// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package load_stats

import (
	// custom import
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	// import from beat generator
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Config stores the config object
type Config struct {
	Ports []string `config:"ports"`
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("connection", "load_stats", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	cfg Config
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config Config

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		cfg:           config,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var err error
	event := mb.Event{}

	for _, port := range m.cfg.Ports {
		event, err = connectionMetric(port)
		if err != nil {
			m.Logger().Errorf("Error getting event for port=%s, error=%s", port, err)
			report.Error(err)
			continue
		}
		isOpen := report.Event(event)
		if !isOpen {
			return nil
		}
	}

	return nil
}

// This func will return event for openvpn connection metric
func connectionMetric(port string) (mb.Event, error) {
	// event send with metrices data
	event := mb.Event{}
	var err error
	network := "tcp"
	timeout := 5 * time.Second
	// connection metric command
	cmd := []byte("load-stats\r\n")
	// connecting on local host in combination of port
	address := net.JoinHostPort("127.0.0.1", port)
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "Connection Error")
	}
	// wirte to telnet
	_, err = conn.Write(cmd)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "Write command Error")
	}
	// listen for reply
	content := readTelnet(conn)
	conn.Close()
	// trim any white space characters \n \r
	content_split := strings.TrimSpace(content)
	stats_fields := strings.Split(content_split, ",")
	nclients, _ := strconv.Atoi(strings.Split(stats_fields[0], "=")[1])
	bytesin, _ := strconv.Atoi(strings.Split(stats_fields[1], "=")[1])
	bytesout, _ := strconv.Atoi(strings.Split(stats_fields[2], "=")[1])
	given_port, _ := strconv.Atoi(port)

	msData := common.MapStr{
		"port":     given_port,
		"clients":  nclients,
		"bytesin":  bytesin,
		"bytesout": bytesout,
	}
	event.MetricSetFields = msData
	return event, nil
}

func readTelnet(conn net.Conn) (out string) {
	var buffer [1]byte
	recvData := buffer[:]
	var err error
	var flag int = 0
	for {
		_, err = conn.Read(recvData)
		if string(recvData) == "\n" {
			flag += 1
		}
		if flag == 2 || err != nil {
			break
		}
		if flag == 1 {
			out += string(recvData)
		}
	}
	return out
}
