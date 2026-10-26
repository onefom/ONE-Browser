//go:build !windows

package browser

func (m *Manager) cleanupManagedExternalExtensionRegistry() error { return nil }
