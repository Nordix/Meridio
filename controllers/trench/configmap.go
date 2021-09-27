package trench

import (
	"fmt"
	"reflect"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	meridioconfig "github.com/nordix/meridio-operator/controllers/config"
	corev1 "k8s.io/api/core/v1"
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

func diffConfigContent(mapA, mapB map[string]string) (bool, string) {
	vip, vipmsg := diffVips(mapA[meridioconfig.VipsConfigKey], mapB[meridioconfig.VipsConfigKey])
	return vip, vipmsg
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

func (c *ConfigMap) listVipsByLabel() (*meridiov1alpha1.VipList, error) {
	vipList := &meridiov1alpha1.VipList{}

	err := c.exec.ListObject(vipList, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return vipList, nil
}

func (c *ConfigMap) listAttractorsByLabel() (*meridiov1alpha1.AttractorList, error) {
	lst := &meridiov1alpha1.AttractorList{}

	err := c.exec.ListObject(lst, client.InNamespace(c.trench.ObjectMeta.Namespace), client.MatchingLabels{"trench": c.trench.ObjectMeta.Name})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return lst, nil
}

func (c *ConfigMap) getDesiredStatus(vl []meridiov1alpha1.Vip) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	configmap.ObjectMeta.Name = common.ConfigMapName(c.trench)
	configmap.ObjectMeta.Namespace = c.trench.ObjectMeta.Namespace

	vdata, err := getVipData(vl)
	if err != nil {
		return nil, err
	}
	configmap.Data = map[string]string{
		meridioconfig.VipsConfigKey: vdata,
	}
	return configmap, nil
}

func (c *ConfigMap) getReconciledDesiredStatus(vl []meridiov1alpha1.Vip, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	vdata, err := getVipData(vl)
	if err != nil {
		return nil, err
	}
	ret := cm.DeepCopy()
	ret.Data[meridioconfig.VipsConfigKey] = vdata
	return ret, nil
}

func getVipData(vips []meridiov1alpha1.Vip) (string, error) {
	config := &meridioconfig.VipConfig{}
	for _, vp := range vips {
		if vp.Status.Status != meridiov1alpha1.Engaged {
			continue
		}
		config.Vips = append(config.Vips, meridioconfig.Vip{
			Name:    vp.ObjectMeta.Name,
			Address: vp.Spec.Address,
		})
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) getAction() ([]common.Action, error) {
	var actions []common.Action
	// get action to update/create the configmap
	cs, err := c.getCurrentStatus()
	if err != nil {
		return nil, err
	}
	elem := common.ConfigMapName(c.trench)

	// get attractors with trench label
	attrs, err := c.listAttractorsByLabel()
	if err != nil {
		return nil, err
	}
	// get vips with trench label
	vips, err := c.listVipsByLabel()
	if err != nil {
		return nil, err
	}
	// make a index map for vip for looking up
	vipIndexMap := func(vips *meridiov1alpha1.VipList) map[string]int {
		im := make(map[string]int)
		for i, v := range vips.Items {
			im[v.ObjectMeta.Name] = i
		}
		return im
	}(vips)

	// for each attractor, search the expected vips exists in the current created vip list or not
	for _, attr := range attrs.Items {
		var vips4attr []meridiov1alpha1.Vip
		for _, v := range attr.Spec.Vips {
			// if index can be found in the index map
			if ind, ok := vipIndexMap[v]; ok {
				vips4attr = append(vips4attr, vips.Items[ind])
			}
		}
		// create configmap if not exist, or update configmap
		if cs == nil {
			c.exec.LogInfo(fmt.Sprintf("waiting %s to be created by attractor", elem))
		} else {
			ds, err := c.getReconciledDesiredStatus(vips4attr, cs)
			if err != nil {
				return nil, err
			}
			if diff, diffmsg := diffConfigContent(cs.Data, ds.Data); diff {
				c.exec.LogInfo(fmt.Sprintf("add action: update %s", elem))
				c.exec.SetOwnerReference(ds, c.trench)
				actions = append(actions, common.NewUpdateAction(ds, fmt.Sprintf("%s, %s", fmt.Sprintf("update %s", elem), diffmsg)))
			}
		}
	}

	return actions, nil
}
