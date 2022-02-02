package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zeebo/xxh3"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	cu "kmodules.xyz/client-go/client"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type UserMenuDriver struct {
	kc   client.Client
	ns   string
	user string
}

func NewUserMenuDriver(kc client.Client, ns, user string) *UserMenuDriver {
	return &UserMenuDriver{
		kc:   kc,
		ns:   ns,
		user: user,
	}
}

func configmapName(user, menu string) string {
	h := xxh3.New()
	if _, err := h.WriteString(user); err != nil {
		panic(errors.Wrapf(err, "failed to hash user %s", user))
	}
	return fmt.Sprintf("%s.%s.%s.%d", rsapi.SchemeGroupVersion.Group, rsapi.ResourceMenuOutline, menu, h.Sum64())
}

//nolint
func getMenuName(user user.Info, cmName string) (string, error) {
	str := strings.TrimSuffix(cmName, "."+user.GetUID())
	idx := strings.LastIndexByte(str, '.')
	if idx == -1 {
		return "", fmt.Errorf("configmap name %s does not match expected menuoutline name format", cmName)
	}
	return str[idx:], nil
}

const (
	keyMenu     = "menu"
	keyUsername = "username"
)

func extractMenu(cm *core.ConfigMap) (*rsapi.MenuOutline, error) {
	data, ok := cm.Data[keyMenu]
	if !ok {
		return nil, apierrors.NewInternalError(fmt.Errorf("ConfigMap %s/%s does not name data[%q]", cm.Namespace, cm.Name, keyMenu))
	}
	var obj rsapi.MenuOutline
	if err := yaml.Unmarshal([]byte(data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func (r *UserMenuDriver) Get(menu string) (*rsapi.MenuOutline, error) {
	cmName := configmapName(r.user, menu)
	var cm core.ConfigMap
	err := r.kc.Get(context.TODO(), client.ObjectKey{Namespace: r.ns, Name: cmName}, &cm)
	if apierrors.IsNotFound(err) {
		return menuoutlines.LoadByName(menu)
	} else if err != nil {
		return nil, err
	}
	return extractMenu(&cm)
}

func (r *UserMenuDriver) List() (*rsapi.MenuOutlineList, error) {
	allMenus := map[string]rsapi.MenuOutline{}
	for _, m := range menuoutlines.List() {
		allMenus[m.Name] = m
	}

	var list core.ConfigMapList
	err := r.kc.List(context.TODO(), &list, client.InNamespace(r.ns), client.MatchingLabels{
		"k8s.io/group":     rsapi.SchemeGroupVersion.Group,
		"k8s.io/kind":      rsapi.ResourceKindMenuOutline,
		"k8s.io/owner-uid": r.user,
	})
	if apierrors.IsNotFound(err) {
		menuoutlines.List()
		return &rsapi.MenuOutlineList{
			TypeMeta: metav1.TypeMeta{},
			// ListMeta: ,
			Items: menuoutlines.List(),
		}, nil
	} else if err != nil {
		return nil, err
	}

	for _, cm := range list.Items {
		menu, err := extractMenu(&cm)
		if err != nil {
			return nil, err
		}
		cmName := configmapName(r.user, menu.Name)
		if cmName != cm.Name {
			return nil, apierrors.NewInternalError(fmt.Errorf("ConfigMap %s/%s contains unexpected menu %s", cm.Namespace, cm.Name, menu.Name))
		}
		allMenus[menu.Name] = *menu
	}

	menus := make([]rsapi.MenuOutline, 0, len(allMenus))
	for _, rl := range allMenus {
		menus = append(menus, rl)
	}
	sort.Slice(menus, func(i, j int) bool {
		return menus[i].Name < menus[j].Name
	})
	return &rsapi.MenuOutlineList{
		TypeMeta: metav1.TypeMeta{},
		// ListMeta: ,
		Items: menus,
	}, nil
}

func (r *UserMenuDriver) Upsert(menu *rsapi.MenuOutline) (*rsapi.MenuOutline, error) {
	data, err := yaml.Marshal(menu)
	if err != nil {
		return nil, apierrors.NewInternalError(errors.Wrapf(err, "failed to marshal MenuOutline %s into yaml", menu.Name))
	}

	var cm core.ConfigMap
	cm.Namespace = r.ns
	cm.Name = configmapName(r.user, menu.Name)
	result, _, err := cu.CreateOrPatch(context.TODO(), r.kc, &cm, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*core.ConfigMap)
		in.Data = map[string]string{
			keyUsername: r.user,
			keyMenu:     string(data),
		}
		return in
	})
	return result.(*rsapi.MenuOutline), err
}
