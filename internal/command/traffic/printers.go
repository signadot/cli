package traffic

import (
	"fmt"
	"io"
)

// printTWProgress prints progress messages during override operations
func printTWProgress(out io.Writer, message string) {
	fmt.Fprintf(out, "→ %s\n", message)
}
