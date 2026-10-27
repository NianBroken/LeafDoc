package preferences

import (
	"os"
	"path/filepath"
)

// RemoveLegacyState removes obsolete private preferences and retired backup credentials.
// Document directories and independent backup archives are never touched.
func RemoveLegacyState(dir string) error {
	for _, name := range []string{"usersettings.db", "usersettings.db-wal", "usersettings.db-shm", "git-backup.json"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
