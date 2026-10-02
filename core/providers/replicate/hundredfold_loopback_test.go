// Modified by Hundredfold AI; see HUNDREDFOLD_MODIFICATIONS.md.

package replicate

import hundredfoldutils "github.com/maximhq/bifrost/core/providers/utils"

// The fork refuses loopback dials unless private networking is enabled, and this
// package's tests dial httptest servers on 127.0.0.1. See
// providers/utils.AllowLoopbackDialsForTesting.
func init() {
	hundredfoldutils.AllowLoopbackDialsForTesting()
}
