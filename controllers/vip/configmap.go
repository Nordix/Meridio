package vip

import (
	"fmt"
	"net"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gopkg.in/yaml.v2"
)

const vipKey = "vips"

type Config struct {
	Vips []Vip `yaml:"items"`
}

type Vip struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

type ConfigMap struct {
	currentStatus *corev1.ConfigMap
	desiredStatus *corev1.ConfigMap
}

func diffVips(cc, cd string) (bool, string) {
	configcd := &Config{}
	err := yaml.Unmarshal([]byte(cc), &configcd)
	if err != nil {
		return true, "Unmarshal desired configmap error"
	}
	configcc := &Config{}
	err = yaml.Unmarshal([]byte(cd), &configcc)
	if err != nil {
		return true, "Unmarshal current configmap error"
	}

	return vipItemsDifferent(configcc.Vips, configcd.Vips)
}

func vipItemsDifferent(vipListA, vipListB []Vip) (bool, string) {
	vipsA := make(map[string]string)
	vipsB := make(map[string]string)
	for _, vip := range vipListA {
		vipsA[vip.Name] = vip.Address
	}
	for _, vip := range vipListB {
		vipsB[vip.Name] = vip.Address
	}
	for name := range vipsA {
		if _, ok := vipsB[name]; !ok {
			return true, fmt.Sprintf("%s needs updated", name)
		}
	}
	for name := range vipsB {
		if _, ok := vipsA[name]; !ok {
			return true, fmt.Sprintf("%s needs updated", name)
		}
	}

	for key, value := range vipsA {
		if vipsB[key] != value {
			return true, fmt.Sprintf("%s needs updated", key)
		}
	}
	return false, ""
}

// func vipItemAddrsEqual(addrListA, addrListB []string) bool {
// 	addrA := make(map[string]struct{})
// 	addrB := make(map[string]struct{})
// 	for _, a := range addrListA {
// 		addrA[a] = struct{}{}
// 	}
// 	for _, a := range addrListB {
// 		addrB[a] = struct{}{}
// 	}
// 	for addr := range addrA {
// 		if _, ok := addrB[addr]; !ok {
// 			return false
// 		}
// 	}
// 	for addr := range addrB {
// 		if _, ok := addrA[addr]; !ok {
// 			return false
// 		}
// 	}
// 	return false
// }

func getTrenchbySelector(e *common.Executor, selector client.ObjectKey) (*meridiov1alpha1.Trench, error) {
	trench := &meridiov1alpha1.Trench{}
	err := e.Client.Get(e.Ctx, selector, trench)
	return trench, err
}

func getConfigMapName(cr *meridiov1alpha1.Trench) string {
	return common.GetFullName(cr, cr.Spec.ConfigMapName)
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
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	c.currentStatus = currentState.DeepCopy()
	return nil
}

func (c *ConfigMap) getDesiredStatus(tv map[string]*net.IPNet, trench *meridiov1alpha1.Trench) error {
	configmap := &corev1.ConfigMap{}
	var err error
	configmap.ObjectMeta.Name = getConfigMapName(trench)
	configmap.ObjectMeta.Namespace = trench.ObjectMeta.Namespace
	data, err := getData(tv)
	if err != nil {
		return err
	}
	configmap.Data = map[string]string{
		vipKey: data,
	}
	c.desiredStatus = configmap
	return nil
}

func (c *ConfigMap) getReconciledDesiredStatus(cm *corev1.ConfigMap, tv map[string]*net.IPNet) error {
	c.desiredStatus = cm.DeepCopy()
	data, err := getData(tv)
	if err != nil {
		return err
	}
	c.desiredStatus.Data[vipKey] = data
	return nil
}

func getData(tv map[string]*net.IPNet) (string, error) {
	config := &Config{}
	for k, vs := range tv {
		config.Vips = append(config.Vips, Vip{
			Name:    k,
			Address: vs.String(),
		})
	}
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error yaml.Marshal: %s", err)
	}
	return string(configYAML), nil
}

func (c *ConfigMap) deleteKey(e *common.Executor, ns, vipName string, tv map[string]map[string]map[string]*net.IPNet) (map[string]map[string]map[string]*net.IPNet, error) {
	// if namespace entry is not found, do nothing
	if _, ok := tv[ns]; !ok {
		e.Log.Info("deleted vip does not have a trench, do nothing")
		return tv, nil
	}
	// find the trench that vip belongs
	var trenchName string
	for tr, value := range tv[ns] {
		if _, ok := value[vipName]; ok {
			trenchName = tr
			break
		}
	}
	// if trench is not found, do nothing
	if trenchName == "" {
		e.Log.Info("deleted vip does not have a trench, do nothing")
		return tv, nil
	}
	// get trench
	selector := client.ObjectKey{
		Namespace: ns,
		Name:      trenchName,
	}
	trench, err := getTrenchbySelector(e, selector)
	if err != nil {
		// if trench is not found
		if apierrors.IsNotFound(err) {
			delete(tv[ns], trenchName)
			return tv, nil
		}
		return tv, err
	}
	// get configmap
	err = c.getCurrentStatus(e.Ctx, trench, e.Client)
	if err != nil {
		return tv, err
	}
	c.desiredStatus = c.currentStatus.DeepCopy()

	delete(tv[ns][trenchName], vipName)
	data, err := getData(tv[ns][trenchName])
	if err != nil {
		return tv, err
	}
	c.desiredStatus.Data[vipKey] = data
	e.Cr = trench
	action := common.NewUpdateAction(c.desiredStatus, fmt.Sprintf("update configmap, deleting vip %s from %s", vipName, trenchName))
	return tv, e.RunAll([]common.Action{action})
}

func (c *ConfigMap) getAction(e *common.Executor, tv map[string]*net.IPNet, vip *meridiov1alpha1.Vip) ([]common.Action, error) {
	var actions []common.Action
	// set the status for the vip
	vipnsname := fmt.Sprintf("%s/%s", vip.GetNamespace(), vip.GetName())
	// if vip is rejected due to trench not founc, update the status only
	actions = append(actions, common.NewUpdateStatusAction(vip, fmt.Sprintf("update vip %s status: %v", vipnsname, vip.Status.Status)))
	if e.Cr == nil {
		return actions, nil
	}
	// if vip is rejected due to overlapping address, also
	actions = append(actions, common.NewUpdateAction(vip, fmt.Sprintf("update vip %s ownerReference", vipnsname)))
	if vip.Status.Status == meridiov1alpha1.PhaseRejected {
		return actions, nil
	}
	// get action to update/create the configmap
	err := c.getCurrentStatus(e.Ctx, e.Cr, e.Client)
	if err != nil {
		return actions, err
	}
	if c.currentStatus == nil {
		err = c.getDesiredStatus(tv, e.Cr)
		if err != nil {
			return actions, err
		}
		msg := fmt.Sprintf("create configmap %s/%s", c.desiredStatus.GetNamespace(), c.desiredStatus.GetName())
		e.Log.Info("configmap", "add action", msg)
		actions = append(actions, common.NewCreateAction(c.desiredStatus, msg))
	} else {
		err = c.getReconciledDesiredStatus(c.currentStatus, tv)
		if err != nil {
			return actions, err
		}
		if diff, diffmsg := diffVips(c.currentStatus.Data[vipKey], c.desiredStatus.Data[vipKey]); diff {
			msg := fmt.Sprintf("update configmap %s/%s", c.desiredStatus.GetNamespace(), c.desiredStatus.GetName())
			e.Log.Info("configmap", "add action", msg)
			actions = append(actions, common.NewUpdateAction(c.desiredStatus, fmt.Sprintf("%s, %s", msg, diffmsg)))
		}
	}
	return actions, nil
}
