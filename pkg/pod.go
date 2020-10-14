package pod

import (
	"context"
	"fmt"
	"log"
	"os"

	guuid "github.com/google/uuid"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodArgs struct {
	PodNamespace string
	Client       *kubernetes.Clientset
	FileID       string
	Input        string
	Output       string
}

// more of a style thing: I consider named returns a bit of an anti-pattern
// explicit returns make it clear when reading the function body what is being returned
func NewPodArgs(fileId, input, output string) (podArgs *PodArgs, err error) {
	podNamespace := os.Getenv("POD_NAMESPACE")

	// the environment never changes after startup. It would be cheaper (and cleaner)
	// to check for this once on start and store the value locally. `os.Getenv` is relatively expensive
	// and we know what the value will be every time
	if podNamespace == "" {
		err = fmt.Errorf("init failed: POD_NAMESPACE environment variable not set")
		return
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	podArgs = &PodArgs{
		PodNamespace: podNamespace,
		Client:       client,
		FileID:       fileId,
		Input:        input,
		Output:       output,
	}
	return
}

func (pa PodArgs) CreatePod() error {
	podSpec := pa.GetPodObject()

	pod, err := pa.Client.CoreV1().Pods(pa.PodNamespace).Create(context.TODO(), podSpec, metav1.CreateOptions{})
	// There are a few reasons why this might fail, including rate limiting
	// I'd suggest retrying this a few time with an exponential backoff
	if err != nil {
		return err
	}

	if pod != nil {
		log.Printf("Successfully created Pod")
	}

	// While unlikely, we should check for the case where pod == nil && err == nil
	// Bugs in libraries do happen...

	return nil
}

// I feel the Pod we spawn should have CPU and memory limits set to prevent a bug or malicious document from
// crashing the node
func (pa PodArgs) GetPodObject() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rebuild-" + guuid.New().String(),
			Namespace: pa.PodNamespace,
		},
		Spec: core.PodSpec{
			ImagePullSecrets: []core.LocalObjectReference{{Name: "regcred"}},
			RestartPolicy:    core.RestartPolicyNever,
			Volumes: []core.Volume{
				{
					Name: "sourcedir",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: "glasswallsource-pvc",
						},
					},
				},
				{
					Name: "targetdir",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: "glasswalltarget-pvc",
						},
					},
				},
			},
			Containers: []core.Container{
				{
					Name:            "rebuild",
					Image:           "glasswallsolutions/icap-request-processing",
					ImagePullPolicy: core.PullIfNotPresent,
					Env: []core.EnvVar{
						{Name: "FILE_ID", Value: pa.FileID},
						{Name: "INPUT_PATH", Value: pa.Input},
						{Name: "OUTPUT_PATH", Value: pa.Output},
					},
					VolumeMounts: []core.VolumeMount{
						{Name: "sourcedir", MountPath: "/input"},
						{Name: "targetdir", MountPath: "/output"},
					},
				},
			},
		},
	}
}
