//go:build windows

package output

// os.File.Sync does not support Windows directory handles. File data is synced
// and manifest replacement uses the same atomic rename boundary as agent state.
func syncDeliveryDirectory(string) error { return nil }
