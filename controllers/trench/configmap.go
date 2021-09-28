package trench

import (
	"fmt"
	"net"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio/pkg/configuration/reader"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gopkg.in/yaml.v2"
)

type ConfigMap struct {
	trench *meridiov1alpha1.Trench
	exec   *common.Executor
}

func NewConfigMap(e *common.Executor, t *meridiov1alpha1.Trench) *ConfigMap {
	l := &ConfigMap{
		trench: t.DeepCopy(),
		exec:   e,
	}
	return l
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

func (c *ConfigMap) listVipsByLabel() (*meridiov1alpha1.VipList, error) {
	vipList := &meridiov1alpha1.VipList{}

	err := c.exec.ListObject(vipList, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return vipList, nil
}

func (c *ConfigMap) listGatewaysByLabel() (*meridiov1alpha1.GatewayList, error) {
	list := &meridiov1alpha1.GatewayList{}

	err := c.exec.ListObject(list, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return list, nil
}

func (c *ConfigMap) listAttractorsByLabel() (*meridiov1alpha1.AttractorList, error) {
	lst := &meridiov1alpha1.AttractorList{}

	err := c.exec.ListObject(lst, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return lst, nil
}

func (c *ConfigMap) getDesiredStatus() (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	configmap.ObjectMeta.Name = common.ConfigMapName(c.trench)
	configmap.ObjectMeta.Namespace = c.trench.ObjectMeta.Namespace

	tdata, err := c.getTrenchData()
	if err != nil {
		return nil, err
	}
	vdata, err := c.getVipData()
	if err != nil {
		return nil, err
	}
	gdata, err := c.getGatewayData()
	if err != nil {
		return nil, err
	}
	attractor, err := c.getAttractorData()
	if err != nil {
		return nil, err
	}
	configmap.Data = map[string]string{
		reader.TrenchConfigKey:     tdata,
		reader.GatewaysConfigKey:   gdata,
		reader.VipsConfigKey:       vdata,
		reader.AttractorsConfigKey: attractor,
	}
	return configmap, nil
}

func (c *ConfigMap) getReconciledDesiredStatus(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	ret := cm.DeepCopy()
	tdata, err := c.getTrenchData()
	if err != nil {
		return nil, err
	}
	ret.Data[reader.TrenchConfigKey] = tdata

	gdata, err := c.getGatewayData()
	if err != nil {
		return nil, err
	}
	ret.Data[reader.GatewaysConfigKey] = gdata

	vdata, err := c.getVipData()
	if err != nil {
		return nil, err
	}
	ret.Data[reader.VipsConfigKey] = vdata

	attractor, err := c.getAttractorData()
	if err != nil {
		return nil, err
	}
	ret.Data[reader.AttractorsConfigKey] = attractor
	return ret, nil
}

func (c *ConfigMap) getTrenchData() (string, error) {
	configYAML, err := yaml.Marshal(&reader.Trench{
		Name: c.trench.ObjectMeta.Name,
	})
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) getVipData() (string, error) {
	// get vips with trench label
	vips, err := c.listVipsByLabel()
	if err != nil {
		return "", err
	}
	config := &reader.VipList{}
	for _, vp := range vips.Items {
		if vp.Status.Status != meridiov1alpha1.Engaged {
			continue
		}
		config.Vips = append(config.Vips, &reader.Vip{
			Name:    vp.ObjectMeta.Name,
			Address: vp.Spec.Address,
			Trench:  c.trench.ObjectMeta.Name,
		})
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) getGatewayData() (string, error) {
	// get gateways with trench label
	gateways, err := c.listGatewaysByLabel()
	if err != nil {
		return "", err
	}
	config := &reader.GatewayList{}
	for _, gw := range gateways.Items {
		if gw.Status.Status != meridiov1alpha1.Engaged {
			continue
		}
		ipFamily := "ipv4"
		if net.ParseIP(gw.Spec.Address).To4() == nil {
			ipFamily = "ipv6"
		}
		ht := parseHoldTime(gw.Spec.Bgp.HoldTime)
		config.Gateways = append(config.Gateways, &reader.Gateway{
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
			Trench:     c.trench.ObjectMeta.Name,
		})
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) getAttractorData() (string, error) {
	// get attractors with trench label
	attrs, err := c.listAttractorsByLabel()
	if err != nil {
		return "", err
	}
	lst := reader.AttractorList{}
	for _, attr := range attrs.Items {
		if attr.Status.LbFe != meridiov1alpha1.Engaged {
			continue
		}
		lst.Attractors = append(lst.Attractors, &reader.Attractor{
			Name:     attr.ObjectMeta.Name,
			Gateways: attr.Spec.Gateways,
			Vips:     attr.Spec.Vips,
			Trench:   c.trench.ObjectMeta.Name,
		})
	}
	configYAML, err := yaml.Marshal(lst)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func parseHoldTime(ht string) uint {
	// validation is done in gateway webhook
	d, _ := time.ParseDuration(ht)
	return uint(d.Seconds())
}

func (c *ConfigMap) getAction() ([]common.Action, error) {
	var actions []common.Action
	// get action to update/create the configmap
	cs, err := c.getCurrentStatus()
	if err != nil {
		return nil, err
	}

	// create configmap if not exist, or update configmap
	if cs == nil {
		ds, err := c.getDesiredStatus()
		if err != nil {
			return nil, err
		}
		actions = append(actions, c.exec.NewCreateAction(ds))
	} else {
		ds, err := c.getReconciledDesiredStatus(cs)
		if err != nil {
			return nil, err
		}
		if !equality.Semantic.DeepEqual(ds.Data, cs.Data) {
			actions = append(actions, c.exec.NewUpdateAction(ds))
		}
	}

	return actions, nil
}
