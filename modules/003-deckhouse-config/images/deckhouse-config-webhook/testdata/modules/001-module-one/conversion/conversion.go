package conversion

import (
	"fmt"

	"github.com/deckhouse/deckhouse/go_lib/deckhouse-config/conversion"
)

const moduleName = "module-one"

var _ = conversion.RegisterFunc(moduleName, 1, 2, convertV1ToV2)

// convertV1ToV2 transforms numeric field to string field.
func convertV1ToV2(values *conversion.JSONValues) error {
	newValue := fmt.Sprintf("%d", values.Get("paramNum").Int())
	_ = values.Delete("paramNum")
	_ = values.Set("paramStr", newValue)
	return nil
}
