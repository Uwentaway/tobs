package volume

import (
	"fmt"

	"github.com/spf13/cobra"
	root "github.com/timescale/tobs/cli/cmd"
	"github.com/timescale/tobs/cli/cmd/common"
	"github.com/timescale/tobs/cli/pkg/helm"
	"github.com/timescale/tobs/cli/pkg/k8s"
)

// volumeGetCmd represents the volume expand command
var volumeGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get PVC's volume sizes",
	Args:  cobra.ExactArgs(0),
	RunE:  volumeGet,
}

var (
	pvcStorage = "storage-volume"
	pvcWAL     = "wal-volume"
)

func init() {
	volumeCmd.AddCommand(volumeGetCmd)
	volumeGetCmd.Flags().BoolP("timescaleDB-wal", "w", false, "Get volume of timescaleDB wal")
	volumeGetCmd.Flags().BoolP("timescaleDB-storage", "s", false, "Get volume of timescaleDB storage")
	volumeGetCmd.Flags().BoolP("prometheus-storage", "p", false, "Get volume of prometheus storage")
}

func volumeGet(cmd *cobra.Command, args []string) error {
	tsDBWal, err := cmd.Flags().GetBool("timescaleDB-wal")
	if err != nil {
		return fmt.Errorf("could not get timescaleDB-wal flag %w", err)
	}

	tsDBStorage, err := cmd.Flags().GetBool("timescaleDB-storage")
	if err != nil {
		return fmt.Errorf("could not get timescaleDB-storage flag %w", err)
	}

	promStorage, err := cmd.Flags().GetBool("prometheus-storage")
	if err != nil {
		return fmt.Errorf("could not get prometheus-storage flag %w", err)
	}

	if !promStorage && !tsDBStorage && !tsDBWal {
		promStorage, tsDBStorage, tsDBWal = true, true, true
	}

	k8sClient := k8s.NewClient()
	if tsDBStorage {
		results, err := k8sClient.GetPVCSizes(root.Namespace, pvcStorage, common.GetTimescaleDBLabels(root.HelmReleaseName))
		if err != nil {
			return fmt.Errorf("could not get timescaleDB-storage: %w", err)
		}
		volumeGetPrint(pvcStorage, results)
	}

	if tsDBWal {
		results, err := k8sClient.GetPVCSizes(root.Namespace, pvcWAL, common.GetTimescaleDBLabels(root.HelmReleaseName))
		if err != nil {
			return fmt.Errorf("could not get timescaleDB-wal: %w", err)
		}
		volumeGetPrint(pvcWAL, results)
	}

	if promStorage {
		pvcPrometheusName, err := pvcPrometheus(root.HelmReleaseName, root.Namespace)
		if err != nil {
			return fmt.Errorf("failed to prometheus pvc name %v", err)
		}
		results, err := k8sClient.GetPVCSizes(root.Namespace, pvcPrometheusName, common.PrometheusLabels)
		if err != nil {
			return fmt.Errorf("could not get prometheus-storage: %w", err)
		}
		volumeGetPrint(pvcPrometheusName, results)
	}

	return nil
}

func volumeGetPrint(pvcPrefix string, results []*k8s.PVCData) {
	if len(results) == 0 {
		return
	}

	fmt.Printf("PVC's of %s\n", pvcPrefix)
	for _, pvc := range results {
		if pvc.SpecSize != pvc.StatusSize {
			fmt.Printf("Existing size of PVC: %s is %s and PVC expansion is in progress to %s\n", pvc.Name, pvc.StatusSize, pvc.SpecSize)
		} else {
			fmt.Printf("Existing size of PVC: %s is %s\n", pvc.Name, pvc.SpecSize)
		}
	}
	fmt.Println()
}

func pvcPrometheus(release, namespace string) (string, error) {
	helmClient := helm.NewClient(namespace)
	defer helmClient.Close()

	name, err := helmClient.ExportValuesFieldFromRelease(release, []string{"kube-prometheus-stack", "fullnameOverride"})
	if err != nil {
		return "", err
	}

	pvcName := "prometheus-" + fmt.Sprint(name) + "-prometheus-db"
	return pvcName, nil
}
