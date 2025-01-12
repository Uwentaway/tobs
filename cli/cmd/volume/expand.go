package volume

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	root "github.com/timescale/tobs/cli/cmd"
	"github.com/timescale/tobs/cli/cmd/common"
	"github.com/timescale/tobs/cli/pkg/k8s"
)

// volumeExpandCmd represents the volume expand command
var volumeExpandCmd = &cobra.Command{
	Use:   "expand",
	Short: "Expand PVC's",
	Args:  cobra.ExactArgs(0),
	RunE:  volumeExpand,
}

func init() {
	volumeCmd.AddCommand(volumeExpandCmd)
	volumeExpandCmd.Flags().StringP("timescaleDB-wal", "w", "", "Expand volume of timescaleDB wal")
	volumeExpandCmd.Flags().StringP("timescaleDB-storage", "s", "", "Expand volume of timescaleDB storage")
	volumeExpandCmd.Flags().StringP("prometheus-storage", "p", "", "Expand volume of prometheus storage")
	volumeExpandCmd.Flags().BoolP("restart-pods", "r", false, "Restarts the pods bound to a PVC on PVC expansion")
	volumeExpandCmd.Flags().BoolP("force-kill", "", false, "On enabling restart-pods this option kills the pods immediately")
	// This flag is hidden as it's only used
	//in tests to force kill pods on restart option
	err := volumeExpandCmd.Flags().MarkHidden("force-kill")
	if err != nil {
		log.Fatal("failed to mark --force-kill flag hidden", err)
	}
}

func volumeExpand(cmd *cobra.Command, args []string) error {
	tsDBWal, err := cmd.Flags().GetString("timescaleDB-wal")
	if err != nil {
		return fmt.Errorf("could not get timescaleDB-wal flag %w", err)
	}

	tsDBStorage, err := cmd.Flags().GetString("timescaleDB-storage")
	if err != nil {
		return fmt.Errorf("could not get timescaleDB-storage flag %w", err)
	}

	promStorage, err := cmd.Flags().GetString("prometheus-storage")
	if err != nil {
		return fmt.Errorf("could not get prometheus-storage flag %w", err)
	}

	restartsPods, err := cmd.Flags().GetBool("restart-pods")
	if err != nil {
		return fmt.Errorf("could not get restart-pods flag %w", err)
	}

	forceKill, err := cmd.Flags().GetBool("force-kill")
	if err != nil {
		return fmt.Errorf("could not get force-kill flag %w", err)
	}

	if promStorage == "" && tsDBStorage == "" && tsDBWal == "" {
		return errors.New("use resource specific flag and provide the desired size for pvc expansion")
	}

	k8sClient := k8s.NewClient()
	if tsDBStorage != "" {
		results, err := k8sClient.ExpandPVCsForAllPods(root.Namespace, tsDBStorage, pvcStorage, common.GetTimescaleDBLabels(root.HelmReleaseName))
		if err != nil {
			return fmt.Errorf("could not expand timescaleDB-storage: %w", err)
		}

		expandSuccessPrint(pvcStorage, results)

		if restartsPods {
			err = restartPods(k8sClient, common.GetTimescaleDBLabels(root.HelmReleaseName), forceKill)
			if err != nil {
				return err
			}
		}
	}

	if tsDBWal != "" {
		results, err := k8sClient.ExpandPVCsForAllPods(root.Namespace, tsDBWal, pvcWAL, common.GetTimescaleDBLabels(root.HelmReleaseName))
		if err != nil {
			return fmt.Errorf("could not expand timescaleDB-wal: %w", err)
		}

		expandSuccessPrint(pvcWAL, results)

		if restartsPods {
			err = restartPods(k8sClient, common.GetTimescaleDBLabels(root.HelmReleaseName), forceKill)
			if err != nil {
				return err
			}
		}
	}

	if promStorage != "" {
		pvcPrometheusName, err := pvcPrometheus(root.HelmReleaseName, root.Namespace)
		if err != nil {
			return fmt.Errorf("failed to prometheus pvc name %v", err)
		}
		results, err := k8sClient.ExpandPVCsForAllPods(root.Namespace, promStorage, pvcPrometheusName, common.PrometheusLabels)
		if err != nil {
			return fmt.Errorf("could not expand prometheus-storage: %w", err)
		}
		expandSuccessPrint(pvcPrometheusName, results)

		if restartsPods {
			err = restartPods(k8sClient, common.PrometheusLabels, forceKill)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func expandSuccessPrint(pvcPrefix string, results map[string]string) {
	if len(results) == 0 {
		return
	}

	fmt.Printf("PVC's of %s\n", pvcPrefix)
	for pvcName, value := range results {
		fmt.Printf("Successfully expanded PVC: %s to %s\n", pvcName, value)
	}
	fmt.Println()
}

func restartPods(k8sClient k8s.Client, labels map[string]string, forceKill bool) error {
	err := k8sClient.DeletePods(root.Namespace, labels, forceKill)
	if err != nil {
		return fmt.Errorf("failed to restart pods after PVC expansion: %w", err)
	}
	fmt.Println("Triggered to restart the pods bound by the PVC's.")
	return nil
}
