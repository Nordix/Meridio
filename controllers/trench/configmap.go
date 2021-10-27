package trench

import (
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

func (c *ConfigMap) listConduitsByLabel() (*meridiov1alpha1.ConduitList, error) {
	lst := &meridiov1alpha1.ConduitList{}

	err := c.exec.ListObject(lst, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return lst, nil
}

func (c *ConfigMap) listStreamsByLabel() (*meridiov1alpha1.StreamList, error) {
	lst := &meridiov1alpha1.StreamList{}

	err := c.exec.ListObject(lst, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return lst, nil
}

func (c *ConfigMap) listFlowsByLabel() (*meridiov1alpha1.FlowList, error) {
	lst := &meridiov1alpha1.FlowList{}

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

	data, err := c.getAllData()
	if err != nil {
		return nil, err
	}
	configmap.Data = data
	return configmap, nil
}

func (c *ConfigMap) getReconciledDesiredStatus(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	ret := cm.DeepCopy()
	data, err := c.getAllData()
	if err != nil {
		return nil, err
	}
	ret.Data = data
	return ret, nil
}

func (c *ConfigMap) getAllData() (map[string]string, error) {
	tdata, err := c.getTrenchData()
	if err != nil {
		return nil, err
	}

	gdata, err := c.getGatewaysData()
	if err != nil {
		return nil, err
	}

	vdata, err := c.getVipsData()
	if err != nil {
		return nil, err
	}

	attractor, err := c.getAttractorsData()
	if err != nil {
		return nil, err
	}

	conduit, err := c.getConduitsData()
	if err != nil {
		return nil, err
	}

	stream, err := c.getStreamsData()
	if err != nil {
		return nil, err
	}

	flow, err := c.getFlowsData()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		reader.TrenchConfigKey:     string(tdata),
		reader.GatewaysConfigKey:   string(gdata),
		reader.VipsConfigKey:       string(vdata),
		reader.AttractorsConfigKey: string(attractor),
		reader.ConduitsConfigKey:   string(conduit),
		reader.StreamsConfigKey:    string(stream),
		reader.FlowsConfigKey:      string(flow),
	}, nil
}

func (c *ConfigMap) getTrenchData() ([]byte, error) {
	return yaml.Marshal(&reader.Trench{
		Name: c.trench.ObjectMeta.Name,
	})
}

func (c *ConfigMap) getVipsData() ([]byte, error) {
	// get vips with trench label
	vips, err := c.listVipsByLabel()
	if err != nil {
		return nil, err
	}
	config := &reader.VipList{}
	for _, vp := range vips.Items {
		if vp.Spec.Address != "" {
			config.Vips = append(config.Vips, &reader.Vip{
				Name:    vp.ObjectMeta.Name,
				Address: vp.Spec.Address,
				Trench:  c.trench.ObjectMeta.Name,
			})
		}
	}

	return yaml.Marshal(config)
}

func (c *ConfigMap) getGatewaysData() ([]byte, error) {
	// get gateways with trench label
	gateways, err := c.listGatewaysByLabel()
	if err != nil {
		return nil, err
	}
	config := &reader.GatewayList{}
	for _, gw := range gateways.Items {
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
	return yaml.Marshal(config)
}

func (c *ConfigMap) getAttractorsData() ([]byte, error) {
	// get attractors with trench label
	attrs, err := c.listAttractorsByLabel()
	if err != nil {
		return nil, err
	}
	lst := reader.AttractorList{}
	for _, attr := range attrs.Items {
		lst.Attractors = append(lst.Attractors, &reader.Attractor{
			Name:     attr.ObjectMeta.Name,
			Gateways: attr.Spec.Gateways,
			Vips:     attr.Spec.Vips,
			Trench:   c.trench.ObjectMeta.Name,
		})
	}
	return yaml.Marshal(lst)
}

func (c *ConfigMap) getConduitsData() ([]byte, error) {
	// get attractors with trench label
	crs, err := c.listConduitsByLabel()
	if err != nil {
		return nil, err
	}
	lst := reader.ConduitList{}
	for _, cr := range crs.Items {
		lst.Conduits = append(lst.Conduits, &reader.Conduit{
			Name:   cr.ObjectMeta.Name,
			Trench: c.trench.ObjectMeta.Name,
		})
	}
	return yaml.Marshal(lst)
}

func (c *ConfigMap) getStreamsData() ([]byte, error) {
	// get attractors with trench label
	crs, err := c.listStreamsByLabel()
	if err != nil {
		return nil, err
	}
	lst := reader.StreamList{}
	for _, cr := range crs.Items {
		// if disengaged or there is not a conduit to sign up yet then skip
		if cr.Spec.Conduit == "" {
			continue
		}
		lst.Streams = append(lst.Streams, &reader.Stream{
			Name:    cr.ObjectMeta.Name,
			Conduit: cr.Spec.Conduit,
		})
	}
	return yaml.Marshal(lst)
}

func (c *ConfigMap) getFlowsData() ([]byte, error) {
	// get attractors with trench label
	crs, err := c.listFlowsByLabel()
	if err != nil {
		return nil, err
	}
	lst := reader.FlowList{}
	for _, cr := range crs.Items {
		// if disengaged or there is not a stream to sign up yet then skip
		if cr.Spec.Stream == "" {
			continue
		}
		lst.Flows = append(lst.Flows, &reader.Flow{
			Name:                  cr.ObjectMeta.Name,
			Stream:                cr.Spec.Stream,
			SourceSubnets:         cr.Spec.SourceSubnets,
			SourcePortRanges:      cr.Spec.SourcePorts,
			DestinationPortRanges: cr.Spec.DestinationPorts,
			Vips:                  cr.Spec.Vips,
			Protocols:             cr.Spec.Protocols,
		})
	}
	return yaml.Marshal(lst)
}

func parseHoldTime(ht string) uint {
	// validation is done in gateway webhook
	d, _ := time.ParseDuration(ht)
	return uint(d.Seconds())
}

func (c *ConfigMap) getAction() error {
	// get action to update/create the configmap
	cs, err := c.getCurrentStatus()
	if err != nil {
		return err
	}

	// create configmap if not exist, or update configmap
	if cs == nil {
		ds, err := c.getDesiredStatus()
		if err != nil {
			return err
		}
		c.exec.AddCreateAction(ds)
	} else {
		ds, err := c.getReconciledDesiredStatus(cs)
		if err != nil {
			return err
		}
		if !equality.Semantic.DeepEqual(ds.Data, cs.Data) {
			c.exec.AddUpdateAction(ds)
		}
	}

	return nil
}
