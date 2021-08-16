package attractor

import (
	"fmt"
	"net"
	"reflect"

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
	vip, vipmsg := diffVips(mapA[meridioconfig.VipsConfigKey], mapB[meridioconfig.VipsConfigKey])
	return gw || vip, fmt.Sprintf("%s %s", gwmsg, vipmsg)
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

func diffVips(cc, cd string) (bool, string) {
	configcd, err := meridioconfig.UnmarshalVipConfig(cd)
	if err != nil {
		return true, fmt.Sprintf("unmarshal desired vip error %s", err)
	}
	mapcd := meridioconfig.MakeMapFromVipList(configcd)
	configcc, err := meridioconfig.UnmarshalVipConfig(cc)
	if err != nil {
		return true, fmt.Sprintf("unmarshal current vip error %s", err)
	}
	mapcc := meridioconfig.MakeMapFromVipList(configcc)
	return vipItemsDifferent(mapcc, mapcd)
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

func vipItemsDifferent(mapA, mapB map[string]meridioconfig.Vip) (bool, string) {
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

func (c *ConfigMap) getDesiredStatus(al *meridiov1alpha1.GatewayList, vl *meridiov1alpha1.VipList) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	configmap.ObjectMeta.Name = common.ConfigMapName(c.trench)
	configmap.ObjectMeta.Namespace = c.trench.ObjectMeta.Namespace

	data, err := c.getGatewayData(al)
	if err != nil {
		return nil, err
	}

	vdata, err := c.getVipData(vl)
	if err != nil {
		return nil, err
	}
	configmap.Data = map[string]string{
		meridioconfig.GatewayConfigKey: data,
		meridioconfig.VipsConfigKey:    vdata,
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

func (c *ConfigMap) listVipsByLabel() (*meridiov1alpha1.VipList, error) {
	vipList := &meridiov1alpha1.VipList{}
	for _, vipName := range c.attr.Spec.Vips {
		vip := &meridiov1alpha1.Vip{}
		err := c.exec.GetObject(client.ObjectKey{
			Name:      vipName,
			Namespace: c.attr.ObjectMeta.Namespace,
		}, vip)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		// referred gateway should also match the attractor label
		sel := labels.Set{"trench": c.trench.ObjectMeta.Name}
		viplbl := vip.ObjectMeta.Labels
		if viplbl == nil {
			viplbl = map[string]string{}
		}
		if labels.SelectorFromSet(sel).Matches(labels.Set(viplbl)) {
			vipList.Items = append(vipList.Items, *vip)
		}
	}
	return vipList, nil
}

func (c *ConfigMap) getReconciledDesiredStatus(al *meridiov1alpha1.GatewayList, vl *meridiov1alpha1.VipList, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	data, err := c.getGatewayData(al)
	if err != nil {
		return nil, err
	}
	vdata, err := c.getVipData(vl)
	if err != nil {
		return nil, err
	}
	ret := cm.DeepCopy()
	ret.Data[meridioconfig.GatewayConfigKey] = data
	ret.Data[meridioconfig.VipsConfigKey] = vdata
	return ret, nil
}

func (c *ConfigMap) getGatewayData(gws *meridiov1alpha1.GatewayList) (string, error) {
	config := &meridioconfig.GatewayConfig{}
	gwlist := []string{}
	for _, gw := range gws.Items {
		if gw.Status.Status != meridiov1alpha1.ConfigStatus.Engaged {
			continue
		}
		ipFamily := "ipv4"
		if net.ParseIP(gw.Spec.Address).To4() == nil {
			ipFamily = "ipv6"
		}
		config.Gateways = append(config.Gateways, meridioconfig.Gateway{
			Name:     gw.ObjectMeta.Name,
			Address:  gw.Spec.Address,
			BFD:      *gw.Spec.BFD,
			Protocol: string(gw.Spec.Protocol),
			IPFamily: ipFamily,
		})
		gwlist = append(gwlist, gw.ObjectMeta.Name)
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	c.attr.Status.GatewayInUse = gwlist
	return string(configYAML), nil
}

func (c *ConfigMap) getVipData(vips *meridiov1alpha1.VipList) (string, error) {
	config := &meridioconfig.VipConfig{}
	viplist := []string{}
	for _, vp := range vips.Items {
		if vp.Status.Status != meridiov1alpha1.ConfigStatus.Engaged {
			continue
		}
		config.Vips = append(config.Vips, meridioconfig.Vip{
			Name:    vp.ObjectMeta.Name,
			Address: vp.Spec.Address,
		})
		viplist = append(viplist, vp.ObjectMeta.Name)
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	c.attr.Status.VipsInUse = viplist
	return string(configYAML), nil
}

func (c *ConfigMap) getAction() (common.Action, error) {
	var action common.Action
	// get action to update/create the configmap
	cs, err := c.getCurrentStatus()
	if err != nil {
		return nil, err
	}
	elem := common.ConfigMapName(c.trench)
	al, err := c.listGatewaysByLabel()
	if err != nil {
		return nil, err
	}
	vl, err := c.listVipsByLabel()
	if err != nil {
		return nil, err
	}

	// update owner of the gateways
	if cs == nil {
		ds, err := c.getDesiredStatus(al, vl)
		if err != nil {
			return nil, err
		}
		c.exec.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds, err := c.getReconciledDesiredStatus(al, vl, cs)
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
