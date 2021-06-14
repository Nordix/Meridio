package configuration

import (
	"context"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Watcher struct {
	configMap        string
	namespace        string
	clientset        *kubernetes.Clientset
	configurationKey string
	configEvent      chan<- *Config
}

type Config struct {
	VIPs []string `yaml:"vips"`
}

func (w *Watcher) Start() {
	for {
		watcher, err := w.clientset.CoreV1().ConfigMaps(w.namespace).Watch(context.TODO(),
			metav1.SingleObject(metav1.ObjectMeta{Name: w.configMap, Namespace: w.namespace}))
		if err != nil {
			logrus.Errorf("Unable to watch configmap: %v", err)
			return
		}
		w.updateCurrentEndpoint(watcher.ResultChan())
	}
}

func (w *Watcher) eventHandler(event *watch.Event) {
	configmap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return
	}
	configuration, ok := configmap.Data[w.configurationKey]
	if !ok {
		return
	}
	config := &Config{}
	err := yaml.Unmarshal([]byte(configuration), &config)
	if err != nil {
		logrus.Errorf("err unmarshal: %v", err)
		return
	}
	logrus.Infof("config: %v", config)
	w.configEvent <- config
}

func (w *Watcher) updateCurrentEndpoint(eventChannel <-chan watch.Event) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				w.eventHandler(&event)
			case watch.Modified:
				w.eventHandler(&event)
			case watch.Deleted:
			default:
			}
		} else {
			return
		}
	}
}

func NewWatcher(configMap string, namespace string, configEvent chan<- *Config) *Watcher {
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		logrus.Errorf("Unable to get InCluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		logrus.Errorf("Unable to create clientset: %v", err)
	}

	watcher := &Watcher{
		configMap:        configMap,
		namespace:        namespace,
		clientset:        clientset,
		configurationKey: "meridio.conf",
		configEvent:      configEvent,
	}
	return watcher
}
