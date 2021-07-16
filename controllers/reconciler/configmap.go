package reconciler

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gopkg.in/yaml.v2"
)

const (
	MeridioConfigKey = "meridio.conf"
)

func getConfigMapName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", cr.Spec.ConfigMapName, cr.ObjectMeta.Name)
}

type Config struct {
	VIPs []string `yaml:"vips"`
}

type ConfigMap struct {
	currentStatus *corev1.ConfigMap
	desiredStatus *corev1.ConfigMap
}

func diffConfigmap(cc *corev1.ConfigMap, cr *meridiov1alpha1.Trench) (bool, string) {
	cd, ok := cc.Data[MeridioConfigKey]
	if !ok {
		return true, "Meridio config key not found"
	}
	config := &Config{}
	err := yaml.Unmarshal([]byte(cd), &config)
	if err != nil {
		return true, "Unmarshal error"
	}
	if !vipListsEqual(config.VIPs, cr.Spec.VIPs) {
		return true, fmt.Sprintf("Different VIPs. Current: %v. Desired: %v", config.VIPs, cr.Spec.VIPs)
	}
	return false, ""
}

func vipListsEqual(vipListA, vipListB []string) bool {
	vipsA := make(map[string]struct{})
	vipsB := make(map[string]struct{})
	for _, vip := range vipListA {
		vipsA[vip] = struct{}{}
	}
	for _, vip := range vipListB {
		vipsB[vip] = struct{}{}
	}
	for _, vip := range vipListA {
		if _, ok := vipsB[vip]; !ok {
			return false
		}
	}
	for _, vip := range vipListB {
		if _, ok := vipsA[vip]; !ok {
			return false
		}
	}
	return true
}

func (c *ConfigMap) getData(trench *meridiov1alpha1.Trench) (string, error) {
	config := &Config{
		VIPs: trench.Spec.VIPs,
	}
	configYAML, err := yaml.Marshal(&config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getConfigMapName(cr),
	}
}

func (c *ConfigMap) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &corev1.ConfigMap{}
	err := client.Get(ctx, c.getSelector(cr), currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	c.currentStatus = currentState.DeepCopy()
	return nil
}

func (c *ConfigMap) insertParamters(cc *corev1.ConfigMap, cr *meridiov1alpha1.Trench) (*corev1.ConfigMap, error) {
	cc.ObjectMeta.Name = getConfigMapName(cr)
	cc.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	data, err := c.getData(cr)
	if err != nil {
		return nil, err
	}
	cc.Data = map[string]string{
		MeridioConfigKey: data,
	}
	return cc, nil
}

func (c *ConfigMap) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	configmap := &corev1.ConfigMap{}
	var err error
	configmap, err = c.insertParamters(configmap, cr)
	if err != nil {
		return err
	}
	c.desiredStatus = configmap
	return nil
}

func (c *ConfigMap) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := c.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return action, err
	}
	if c.currentStatus == nil {
		err = c.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.log.Info("configmap", "add action", "create")
		action = newCreateAction(c.desiredStatus, "create configmap")
	} else {
		if diff, msg := diffConfigmap(c.currentStatus, cr); diff {
			e.log.Info("configmap", "add action", "update")
			action = newUpdateAction(c.desiredStatus, fmt.Sprintf("update configmap, %s", msg))
		}
	}
	return action, nil
}
