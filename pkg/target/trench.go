package target

import (
	"context"
	"fmt"

	"github.com/nordix/meridio/pkg/configuration"
	"github.com/sirupsen/logrus"
)

type Trench struct {
	context              context.Context
	cancel               context.CancelFunc
	name                 string
	namespace            string
	conduits             []*Conduit
	vips                 []string
	configurationWatcher *configuration.OperatorWatcher
	configWatcher        <-chan *configuration.OperatorConfig
	config               *Config
}

func (t *Trench) AddConduit(name string) (*Conduit, error) {
	conduit, err := NewConduit(name, t)
	if err != nil {
		return nil, err
	}
	conduit.SetVIPs(t.vips)
	t.conduits = append(t.conduits, conduit)
	return conduit, nil
}

func (t *Trench) DeleteConduit(name string) error {
	for index, conduit := range t.conduits {
		if conduit.name == name {
			t.conduits = append(t.conduits[:index], t.conduits[index+1:]...)
			return conduit.Delete()
		}
	}
	return nil
}

func (t *Trench) GetConduit(name string) *Conduit {
	for _, conduit := range t.conduits {
		if conduit.name == name {
			return conduit
		}
	}
	return nil
}

func (t *Trench) GetConduits() []*Conduit {
	return t.conduits
}

func (t *Trench) Delete() error {
	t.cancel()
	t.configurationWatcher.Delete()
	for _, conduit := range t.conduits {
		err := conduit.Delete()
		if err != nil {
			logrus.Errorf("Error deleting a conduit: %v", err)
		}
	}
	return nil
}

func (t *Trench) watchConfig() {
	for {
		select {
		case config := <-t.configWatcher:
			t.vips = configuration.AddrListFromVipConfig(config.VIPs)
			for _, conduit := range t.conduits {
				conduit.SetVIPs(t.vips)
			}
		case <-t.context.Done():
			return
		}
	}
}

func (t *Trench) GetName() string {
	return t.name
}

func (t *Trench) GetNamespace() string {
	return t.namespace
}

func (t *Trench) GetConfig() *Config {
	return t.config
}

func NewTrench(name string, namespace string, config *Config) *Trench {
	configMapName := fmt.Sprintf("%s-%s", config.configMapName, name)
	configWatcher := make(chan *configuration.OperatorConfig)
	configurationWatcher := configuration.NewOperatorWatcher(configMapName, namespace, configWatcher)
	go configurationWatcher.Start()

	context, cancel := context.WithCancel(context.Background())

	trench := &Trench{
		context:              context,
		cancel:               cancel,
		name:                 name,
		namespace:            namespace,
		conduits:             []*Conduit{},
		vips:                 []string{},
		configurationWatcher: configurationWatcher,
		configWatcher:        configWatcher,
		config:               config,
	}

	go trench.watchConfig()

	return trench
}
