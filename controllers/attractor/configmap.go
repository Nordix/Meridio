package attractor

import (
	"fmt"
	"net"
	"reflect"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	meridioconfig "github.com/nordix/meridio-operator/controllers/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gopkg.in/yaml.v2"
)

type ConfigMap struct {
	attr   *meridiov1alpha1.Attractor
	trench *meridiov1alpha1.Trench
	exec   *common.Executor
}

func NewConfigMap(e *common.Executor, t *meridiov1alpha1.Trench, attr *meridiov1alpha1.Attractor) *ConfigMap {
	l := &ConfigMap{
		trench: t.DeepCopy(),
		attr:   attr,
		exec:   e,
	}
	return l
}

func diffConfigContent(mapA, mapB map[string]string) (bool, string) {
	gw, gwmsg := diffGateways(mapA[meridioconfig.GatewayConfigKey], mapB[meridioconfig.GatewayConfigKey])
	return gw, gwmsg
}

func diffGateways(cc, cd string) (bool, string) {
	configcd, err := meridioconfig.UnmarshalGatewayConfig(cd)
	if err != nil {
		return true, fmt.Sprintf("unmarshal desired gateway error %s", err)
	}
	mapcd := meridioconfig.MakeMapFromGWList(configcd)
	configcc, err := meridioconfig.UnmarshalGatewayConfig(cc)
	if err != nil {
		return true, fmt.Sprintf("unmarshal current gateway error %s", err)
	}
	mapcc := meridioconfig.MakeMapFromGWList(configcc)
	return gwItemsDifferent(mapcc, mapcd)
}

func gwItemsDifferent(mapA, mapB map[string]meridioconfig.Gateway) (bool, string) {
	for name := range mapA {
		if _, ok := mapB[name]; !ok {
			return true, fmt.Sprintf("%s needs updated", name)
		}
	}
	for name := range mapB {
		if _, ok := mapA[name]; !ok {
			return true, fmt.Sprintf("%s needs updated", name)
		}
	}
	for key, value := range mapA {
		if !reflect.DeepEqual(mapB[key], value) {
			return true, fmt.Sprintf("%s needs updated", key)
		}
	}
	return false, ""
}

func (c *ConfigMap) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: c.trench.ObjectMeta.Namespace,
		Name:      common.ConfigMapName(c.trench),
	}
}

func (c *ConfigMap) getCurrentStatus() (*corev1.ConfigMap, error) {
	currentState := &corev1.ConfigMap{}
	err := c.exec.GetObject(c.getSelector(), currentState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentState, nil
}

func (c *ConfigMap) getDesiredStatus(al *meridiov1alpha1.GatewayList) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	configmap.ObjectMeta.Name = common.ConfigMapName(c.trench)
	configmap.ObjectMeta.Namespace = c.trench.ObjectMeta.Namespace

	data, err := c.getGatewayData(al)
	if err != nil {
		return nil, err
	}

	configmap.Data = map[string]string{
		meridioconfig.GatewayConfigKey: data,
	}
	return configmap, nil
}

// list all existing gateways expected by attractor
func (c *ConfigMap) listGatewaysByLabel() (*meridiov1alpha1.GatewayList, error) {
	gatewayList := &meridiov1alpha1.GatewayList{}
	for _, gwName := range c.attr.Spec.Gateways {
		// iterating 'gateway' field in attractor, find gateway by name
		gateway := &meridiov1alpha1.Gateway{}
		err := c.exec.GetObject(client.ObjectKey{
			Name:      gwName,
			Namespace: c.attr.ObjectMeta.Namespace,
		}, gateway)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		// only append the gateways having the attractor label same as this attractor to the return gateway list
		sel := labels.Set{
			"attractor": c.attr.ObjectMeta.Name,
		}
		gatewayLabels := gateway.ObjectMeta.Labels
		if gatewayLabels == nil {
			gatewayLabels = map[string]string{}
		}
		if labels.SelectorFromSet(sel).Matches(labels.Set(gatewayLabels)) {
			gatewayList.Items = append(gatewayList.Items, *gateway)
		}
	}
	return gatewayList, nil
}

func (c *ConfigMap) getReconciledDesiredStatus(al *meridiov1alpha1.GatewayList, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	data, err := c.getGatewayData(al)
	if err != nil {
		return nil, err
	}
	ret := cm.DeepCopy()
	ret.Data[meridioconfig.GatewayConfigKey] = data
	return ret, nil
}

func (c *ConfigMap) getGatewayData(gws *meridiov1alpha1.GatewayList) (string, error) {
	config := &meridioconfig.GatewayConfig{}
	for _, gw := range gws.Items {
		if gw.Status.Status != meridiov1alpha1.Engaged {
			continue
		}
		ipFamily := "ipv4"
		if net.ParseIP(gw.Spec.Address).To4() == nil {
			ipFamily = "ipv6"
		}
		ht := parseHoldTime(gw.Spec.Bgp.HoldTime)
		config.Gateways = append(config.Gateways, meridioconfig.Gateway{
			Name:       gw.ObjectMeta.Name,
			Address:    gw.Spec.Address,
			BFD:        *gw.Spec.Bgp.BFD,
			Protocol:   string(gw.Spec.Protocol),
			RemoteASN:  *gw.Spec.Bgp.RemoteASN,
			LocalASN:   *gw.Spec.Bgp.LocalASN,
			RemotePort: *gw.Spec.Bgp.RemotePort,
			LocalPort:  *gw.Spec.Bgp.LocalPort,
			HoldTime:   ht,
			IPFamily:   ipFamily,
		})
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

// Update attractor status.gateway-in-use
func (c *ConfigMap) getGatewayInUse(cm *corev1.ConfigMap) error {
	lst := []string{}
	gw, err := meridioconfig.UnmarshalGatewayConfig(cm.Data[meridioconfig.GatewayConfigKey])
	if err != nil {
		return fmt.Errorf("unmarshal gateway error %s", err)
	}
	mapgw := meridioconfig.MakeMapFromGWList(gw)
	for key := range mapgw {
		lst = append(lst, key)
	}
	c.attr.Status.GatewayInUse = lst
	return nil
}

// Update attractor status.vips-in-use
func (c *ConfigMap) getVipsInUse(cm *corev1.ConfigMap) error {
	lst := []string{}
	vip, err := meridioconfig.UnmarshalVipConfig(cm.Data[meridioconfig.VipsConfigKey])
	if err != nil {
		return fmt.Errorf("unmarshal vip error %s", err)
	}
	mapvp := meridioconfig.MakeMapFromVipList(vip)
	for key := range mapvp {
		lst = append(lst, key)
	}
	c.attr.Status.VipsInUse = lst
	return nil
}

func parseHoldTime(ht string) uint {
	// validation is done in gateway webhook
	d, _ := time.ParseDuration(ht)
	return uint(d.Seconds())
}

func (c *ConfigMap) getAction() (common.Action, error) {
	var action common.Action
	// get action to update/create the configmap
	cs, err := c.getCurrentStatus()
	if err != nil {
		return nil, err
	}
	// update attractor status according to the configmap
	if cs != nil {
		err = c.getGatewayInUse(cs)
		if err != nil {
			return nil, fmt.Errorf("setting attractor status.gateway-in-use error %s", err)
		}
		err = c.getVipsInUse(cs)
		if err != nil {
			return nil, fmt.Errorf("setting attractor status.vips-in-use error %s", err)
		}
	}

	// update configmap
	elem := common.ConfigMapName(c.trench)
	al, err := c.listGatewaysByLabel()
	if err != nil {
		return nil, err
	}
	if cs == nil {
		ds, err := c.getDesiredStatus(al)
		if err != nil {
			return nil, err
		}
		c.exec.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds, err := c.getReconciledDesiredStatus(al, cs)
		if err != nil {
			return nil, err
		}
		if diff, diffmsg := diffConfigContent(cs.Data, ds.Data); diff {
			c.exec.LogInfo(fmt.Sprintf("add action: update %s", elem))
			action = common.NewUpdateAction(ds, fmt.Sprintf("%s, %s", fmt.Sprintf("update %s", elem), diffmsg))
		}
	}
	return action, nil
}
