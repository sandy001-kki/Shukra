// This file contains shared naming and helper logic for child resource builders.
// It exists so every builder uses deterministic names and consistent labels.
package resources

import (
	"fmt"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func Name(appEnv *appsv1beta1.AppEnvironment, suffix string) string {
	return fmt.Sprintf("%s-%s", appEnv.Name, suffix)
}
