package main

import (
	"encoding/gob"
	"fmt"
	"path/filepath"

	cu "kmodules.xyz/client-go/client"

	flag "github.com/spf13/pflag"
	"github.com/zeebo/xxh3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

/*
const (
	MenuAccordion MenuMode = "Accordion"
	MenuDropDown  MenuMode = "DropDown"
	MenuGallery   MenuMode = "Gallery"
)
*/

var (
	masterURL      = ""
	kubeconfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")

	url     = "https://raw.githubusercontent.com/kubepack/preset-testdata/master/stable/"
	name    = "hello"
	version = "0.1.0"
)

func main_hash() {
	m := "meta.k8s.appscode.com.menuoutline.cluster.system:serviceaccount:kube-system:lke-admin"
	fmt.Printf("%v\n", HashMessage(m))

	h := xxh3.New()
	_, _ = h.WriteString(m)
	fmt.Println(h.Sum64())
}

func HashMessage(m string) interface{} {
	h := xxh3.New()
	if err := gob.NewEncoder(h).Encode(m); err != nil {
		panic(fmt.Errorf("failed to gob encode %#v: %w", m, err))
	}

	return h.Sum64()
}

func main() {
	flag.StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	flag.StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&url, "url", url, "Chart repo url")
	flag.StringVar(&name, "name", name, "Name of bundle")
	flag.StringVar(&version, "version", version, "Version of bundle")
	flag.Parse()

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterURL}})
	cfg, err := cc.ClientConfig()
	if err != nil {
		klog.Fatal(err)
	}

	client := kubernetes.NewForConfigOrDie(cfg)
	kc, err := cu.NewUncachedClient(cfg)
	if err != nil {
		klog.Fatal(err)
	}

	out, err := RenderAccordionMenu(kc, client, "kubedb")
	data, _ := yaml.Marshal(out)
	fmt.Println(string(data))

	fmt.Println("--------------------------------")

	out2, err := RenderAccordionMenu(kc, client, "kubedb")
	data2, _ := yaml.Marshal(out2)
	fmt.Println(string(data2))

	//menu, err := GenerateCompleteMenu(kc, client.Discovery())
	//if err != nil {
	//	klog.Fatal(err)
	//}
	//data, _ := yaml.Marshal(menu)
	//fmt.Println(string(data))

	//in := &rsapi.RenderMenuRequest{
	//	Menu:    "cluster",
	//	Mode:    rsapi.MenuGallery,
	//	Section: nil,
	//	Type:    nil,
	//}
	//in := &rsapi.RenderMenuRequest{
	//	Menu:    "cluster",
	//	Mode:    rsapi.MenuAccordion,
	//	Section: nil,
	//	Type:    nil,
	//}
	//in := &rsapi.RenderMenuRequest{
	//	Menu:    "cluster",
	//	Mode:    rsapi.MenuDropDown,
	//	Section: pointer.StringP("Workloads"),
	//	Type:    nil,
	//}
	//driver := NewUserMenuDriver(kc, client, "default", "")
	//out, err := RenderMenu(driver, in)
	//if err != nil {
	//	klog.Fatal(err)
	//}
	//data2, _ := yaml.Marshal(out)
	//fmt.Println(string(data2))
}
