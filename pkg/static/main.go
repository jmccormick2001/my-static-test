// Note: the example only works with the code within the same release/branch.
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/api/pkg/manifests"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	version = "0.0.1"
)

func main() {
	fmt.Printf("my-static-test version is %s\n", version)

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var config *rest.Config
	var err error

	if fileExists(*kubeconfig) {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {

		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	configMapName := os.Getenv("CONFIGMAP_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	err = extractBundle(clientset, configMapName, namespace)
	if err != nil {
		panic(err.Error())
	}

	bundlePath := "/tmp/output-folder"

	files, err := Unzip("/tmp/bundle.zip", bundlePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))

	_, _, validationResults := manifests.GetManifestsDir(bundlePath)
	for _, result := range validationResults {
		for _, e := range result.Errors {
			fmt.Printf("Error: %s\n", e)
		}

		for _, w := range result.Warnings {
			fmt.Printf("Warning: %s\n", w.Error())
		}
	}

}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

// getConfigMap gets a ConfigMap by name
func getConfigMap(clientset *kubernetes.Clientset, name, namespace string) (*v1.ConfigMap, error) {
	cfg, err := clientset.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		fmt.Println(err)
		return cfg, err
	}

	return cfg, nil
}

func extractBundle(clientset *kubernetes.Clientset, configMapName, namespace string) error {
	configMap, err := getConfigMap(clientset, configMapName, namespace)
	if err != nil {
		fmt.Printf("could not get configmap %s\n", configMapName)
		return err
	}

	b := configMap.BinaryData["bundle"]
	if len(b) == 0 {
		fmt.Println("could not get configmap binarydata")
		return err
	}
	zipFileContents := []byte(b)

	file, err := os.OpenFile(
		"/tmp/bundle.zip",
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	bytesWritten, err := file.Write(zipFileContents)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("wrote %d bytes to /tmp/bundle.zip\n", bytesWritten)
	return nil

}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
