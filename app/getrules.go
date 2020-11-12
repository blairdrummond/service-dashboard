/*  Scrape the ingress rules in this app's namespace for
*   all backend services and paths.
*
*   Then search to see if there is a config-map associated
*   to that backend - that's where we're storing the
*   $maintainer and $description info.
*
*   NOTE: Relies on the environment variable NAMESPACE
 */

package main

import (
	"context"
	// "flag"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const default_description = "No Description Found"
const default_maintainer = "No maintainer listed"

type UserService struct {
	Name        string
	Description string
	URL         string
	Maintainer  string
}

func GetClient() (*k8s.Clientset, error) {

	/*
	* // FOR LOCAL TESTING
	* // use the current context in kubeconfig
	* var kubeconfig *string
	* if home := homedir.HomeDir(); home != "" {
	* 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	* } else {
	* 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	* }
	* flag.Parse()
	* config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	 */

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	return k8s.NewForConfig(config)
}

/* Search for a configmap containing the app metadata.
*
*  Return the UserService s, but with the maintainer and
*  description info added (or a default if not found/available.)
 */
func AddMetaData(clientset *k8s.Clientset, s UserService) UserService {

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": s.Name}}

	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		Limit:         100,
	}

	namespace := os.Getenv("NAMESPACE")
	configMapList, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), listOptions)

	if err != nil {
		fmt.Printf("There was an error trying to find the ConfigMap for %s\n", s.Name)
		fmt.Printf("%s\n", err.Error())

		// Return the default instead.
		return UserService{s.Name, default_description, s.URL, default_maintainer}
	}

	items := configMapList.Items
	if len(items) > 1 {
		fmt.Printf("WARNING: There was more than one ConfigMap for %s\n", s.Name)
	} else if len(items) == 0 {
		fmt.Printf("No ConfigMap found for %s\n", s.Name)
		return UserService{s.Name, default_description, s.URL, default_maintainer}
	} else {
		fmt.Printf("Found ConfigMap for %s\n", s.Name)
	}

	cf := items[0].Data

	// Defaults
	description := default_description
	maintainer := default_maintainer

	// Extract the data from the configmap
	if val, ok := cf["maintainer"]; ok {
		maintainer = val
	}

	if val, ok := cf["description"]; ok {
		description = val
	}

	return UserService{s.Name, description, s.URL, maintainer}

}

/*  # Curl command for reference
*   curl -s --user admin:dcfa2bad0c492539ea83a7c1ca546f85 --insecure https://10.0.0.249:6443/apis/extensions/v1beta1/namespaces/web/ingresses
*      | jq -c '.items
*               | .[]
*               | .spec.rules
*               | .[]
*               | select(.host | startswith("www."))
*               | .http.paths
*               | .[]
*               | { "service" : .backend.serviceName, "path" : .path }'
*
 */

/*  Scrape the ingress rules in this app's namespace for
*   all backend services and paths.
*
*   Then search to see if there is a config-map associated
*   to that backend - that's where we're storing the
*   $maintainer and $description info.
 */
func GetApps(clientset *k8s.Clientset) []UserService {

	// This has to get set in the chart
	namespace := os.Getenv("NAMESPACE")

	// Get ingresses
	ingressList, err := clientset.ExtensionsV1beta1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d ingress-items in the cluster\n", len(ingressList.Items))

	var serviceList []UserService
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.IngressRuleValue.HTTP.Paths {
				backend := path.Backend.ServiceName

				// Skip the root path! I.e. the dashboard itself.
				if path.Path == "/" {
					continue
				}

				path_str := fmt.Sprintf("%s%s", rule.Host, path.Path)

				app := UserService{backend, "", path_str, ""}
				serviceList = append(serviceList, app)
				fmt.Printf("Found: %s\n", app)
			}
		}
	}

	// Take one pass through the list and add all metadata
	for i, s := range serviceList {
		serviceList[i] = AddMetaData(clientset, s)
	}

	//time.Sleep(10 * time.Second)
	return serviceList[:]
}
