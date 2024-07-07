/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	maxFileNameLength = 255
	maxPathLength     = 4096
)

// AtomicWriter handles atomically projecting content for a set of files into
// a target directory.
//
// Note:
//
//  1. AtomicWriter reserves the set of path names starting with `..`.
//  2. AtomicWriter offers no concurrency guarantees and must be synchronized
//     by the caller.
//
// The visible files in this volume are symlinks to files in the writer's data
// directory.  Actual files are stored in a hidden timestamped directory which
// is symlinked to by the data directory. The timestamped directory and
// data directory symlink are created in the writer's target dir.  This scheme
// allows the files to be atomically updated by changing the target of the
// data directory symlink.
//
// Consumers of the target directory can monitor the ..data symlink using
// inotify or fsnotify to receive events when the content in the volume is
// updated.
type AtomicWriter struct {
	targetDir string
}

// FileProjection contains file Data and access Mode
type FileProjection struct {
	Data    []byte
	Mode    int32
	FsUser  *int64
	FsGroup *int64
}

// NewAtomicWriter creates a new AtomicWriter configured to write to the given
// target directory, or returns an error if the target directory does not exist.
func NewAtomicWriter(targetDir string) (*AtomicWriter, error) {
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		return nil, err
	}

	return &AtomicWriter{targetDir: targetDir}, nil
}

const (
	dataDirName    = "..data"
	newDataDirName = "..data_tmp"
)

// Write does an atomic projection of the given payload into the writer's target
// directory.  Input paths must not begin with '..'.
// setPerms is an optional pointer to a function that caller can provide to set the
// permissions of the newly created files before they are published. The function is
// passed subPath which is the name of the timestamped directory that was created
// under target directory.
//
// The Write algorithm is:
//
//  1. The payload is validated; if the payload is invalid, the function returns
//
//  2. The current timestamped directory is detected by reading the data directory
//     symlink
//
//  3. The old version of the volume is walked to determine whether any
//     portion of the payload was deleted and is still present on disk.
//
//  4. The data in the current timestamped directory is compared to the projected
//     data to determine if an update to data directory is required.
//
//  5. A new timestamped dir is created if an update is required.
//
//  6. The payload is written to the new timestamped directory.
//
//  7. Permissions are set (if setPerms is not nil) on the new timestamped directory and files.
//
//  8. A symlink to the new timestamped directory ..data_tmp is created that will
//     become the new data directory.
//
//  9. The new data directory symlink is renamed to the data directory; rename is atomic.
//
//  10. Symlinks and directory for new user-visible files are created (if needed).
//
//     For example, consider the files:
//     <target-dir>/podName
//     <target-dir>/user/labels
//     <target-dir>/k8s/annotations
//
//     The user visible files are symbolic links into the internal data directory:
//     <target-dir>/podName         -> ..data/podName
//     <target-dir>/usr -> ..data/usr
//     <target-dir>/k8s -> ..data/k8s
//
//     The data directory itself is a link to a timestamped directory with
//     the real data:
//     <target-dir>/..data          -> ..2016_02_01_15_04_05.12345678/
//     NOTE(claudiub): We need to create these symlinks AFTER we've finished creating and
//     linking everything else. On Windows, if a target does not exist, the created symlink
//     will not work properly if the target ends up being a directory.
//
//  11. Old paths are removed from the user-visible portion of the target directory.
//
//  12. The previous timestamped directory is removed, if it exists.
func (w *AtomicWriter) Write(ctx context.Context, payload map[string]FileProjection, setPerms func(subPath string) error) error {
	logger := log.FromContext(ctx)

	// (1)
	cleanPayload, err := validatePayload(payload)
	if err != nil {
		logger.Error(err, "invalid payload")
		return err
	}

	// (2)
	dataDirPath := filepath.Join(w.targetDir, dataDirName)
	oldTsDir, err := os.Readlink(dataDirPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Error(err, "error reading link for data directory")
			return err
		}
		// although Readlink() returns "" on err, don't be fragile by relying on it (since it's not specified in docs)
		// empty oldTsDir indicates that it didn't exist
		oldTsDir = ""
	}
	oldTsPath := filepath.Join(w.targetDir, oldTsDir)

	var pathsToRemove sets.Set[string]
	shouldWrite := true
	// if there was no old version, there's nothing to remove
	if len(oldTsDir) != 0 {
		// (3)
		pathsToRemove, err = w.pathsToRemove(logger, cleanPayload, oldTsPath)
		if err != nil {
			logger.Error(err, "error determining user-visible files to remove")
			return err
		}

		// (4)
		if should, err := shouldWritePayload(cleanPayload, oldTsPath); err != nil {
			logger.Error(err, "error determining whether payload should be written to disk")
			return err
		} else if !should && len(pathsToRemove) == 0 {
			logger.V(4).Info("write not required for data directory", "dir", oldTsDir)
			// data directory is already up to date, but we need to make sure that
			// the user-visible symlinks are created.
			// See https://github.com/kubernetes/kubernetes/issues/121472 for more details.
			// Reset oldTsDir to empty string to avoid removing the data directory.
			shouldWrite = false
			oldTsDir = ""
		} else {
			logger.V(4).Info("write required for target directory", "dir", w.targetDir)
		}
	}

	if shouldWrite {
		// (5)
		tsDir, err := w.newTimestampDir(logger)
		if err != nil {
			logger.Error(err, "error creating new ts data director")
			return err
		}
		tsDirName := filepath.Base(tsDir)

		// (6)
		if err = w.writePayloadToDir(logger, cleanPayload, tsDir); err != nil {
			logger.Error(err, "error writing payload to ts data directory", "dir", tsDir)
			return err
		}
		logger.V(4).Info("performed write of new data to ts data directory", "dir", w.targetDir)

		// (7)
		if setPerms != nil {
			if err := setPerms(tsDirName); err != nil {
				logger.Error(err, "error applying ownership settings")
				return err
			}
		}

		// (8)
		newDataDirPath := filepath.Join(w.targetDir, newDataDirName)
		if err = os.Symlink(tsDirName, newDataDirPath); err != nil {
			if err := os.RemoveAll(tsDir); err != nil {
				logger.Error(err, "error removing new ts directory", "dir", tsDir)
			}
			logger.Error(err, "error creating symbolic link for atomic update")
			return err
		}

		// (9)
		if runtime.GOOS == "windows" {
			if err := os.Remove(dataDirPath); err != nil {
				logger.Error(err, "error removing data dir directory", "dir", dataDirPath)
			}
			err = os.Symlink(tsDirName, dataDirPath)
			if err := os.Remove(newDataDirPath); err != nil {
				logger.Error(err, "error removing new data dir directory", "dir", newDataDirPath)
			}
		} else {
			err = os.Rename(newDataDirPath, dataDirPath)
		}
		if err != nil {
			if err := os.Remove(newDataDirPath); err != nil && err != os.ErrNotExist {
				logger.Error(err, "error removing new data dir directory", "dir", newDataDirPath)
			}
			if err := os.RemoveAll(tsDir); err != nil {
				logger.Error(err, "error removing new ts directory", "dir", tsDir)
			}
			logger.Error(err, "error renaming symbolic link for data director", "dir", newDataDirPath)
			return err
		}
	}

	// (10)
	if err = w.createUserVisibleFiles(cleanPayload); err != nil {
		logger.Error(err, "error creating visible symlinks", "dir", w.targetDir)
		return err
	}

	// (11)
	if err = w.removeUserVisiblePaths(logger, pathsToRemove); err != nil {
		logger.Error(err, "error removing old visible symlinks")
		return err
	}

	// (12)
	if len(oldTsDir) > 0 {
		if err = os.RemoveAll(oldTsPath); err != nil {
			logger.Error(err, "error removing old data directory", "dir", oldTsDir)
			return err
		}
	}

	return nil
}

// validatePayload returns an error if any path in the payload returns a copy of the payload with the paths cleaned.
func validatePayload(payload map[string]FileProjection) (map[string]FileProjection, error) {
	cleanPayload := make(map[string]FileProjection)
	for k, content := range payload {
		if err := validatePath(k); err != nil {
			return nil, err
		}

		cleanPayload[filepath.Clean(k)] = content
	}

	return cleanPayload, nil
}

// validatePath validates a single path, returning an error if the path is
// invalid.  paths may not:
//
// 1. be absolute
// 2. contain '..' as an element
// 3. start with '..'
// 4. contain filenames larger than 255 characters
// 5. be longer than 4096 characters
func validatePath(targetPath string) error {
	// TODO: somehow unify this with the similar api validation,
	// validateVolumeSourcePath; the error semantics are just different enough
	// from this that it was time-prohibitive trying to find the right
	// refactoring to re-use.
	if targetPath == "" {
		return fmt.Errorf("invalid path: must not be empty: %q", targetPath)
	}
	if path.IsAbs(targetPath) {
		return fmt.Errorf("invalid path: must be relative path: %s", targetPath)
	}

	if len(targetPath) > maxPathLength {
		return fmt.Errorf("invalid path: must be less than or equal to %d characters", maxPathLength)
	}

	items := strings.Split(targetPath, string(os.PathSeparator))
	for _, item := range items {
		if item == ".." {
			return fmt.Errorf("invalid path: must not contain '..': %s", targetPath)
		}
		if len(item) > maxFileNameLength {
			return fmt.Errorf("invalid path: filenames must be less than or equal to %d characters", maxFileNameLength)
		}
	}
	if strings.HasPrefix(items[0], "..") && len(items[0]) > 2 {
		return fmt.Errorf("invalid path: must not start with '..': %s", targetPath)
	}

	return nil
}

// shouldWritePayload returns whether the payload should be written to disk.
func shouldWritePayload(payload map[string]FileProjection, oldTsDir string) (bool, error) {
	for userVisiblePath, fileProjection := range payload {
		shouldWrite, err := shouldWriteFile(filepath.Join(oldTsDir, userVisiblePath), fileProjection.Data)
		if err != nil {
			return false, err
		}

		if shouldWrite {
			return true, nil
		}
	}

	return false, nil
}

// shouldWriteFile returns whether a new version of a file should be written to disk.
func shouldWriteFile(path string, content []byte) (bool, error) {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return true, nil
	}

	contentOnFs, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	return !bytes.Equal(content, contentOnFs), nil
}

// pathsToRemove walks the current version of the data directory and
// determines which paths should be removed (if any) after the payload is
// written to the target directory.
func (w *AtomicWriter) pathsToRemove(logger logr.Logger, payload map[string]FileProjection, oldTSDir string) (sets.Set[string], error) {
	paths := sets.New[string]()
	visitor := func(path string, info os.FileInfo, err error) error {
		relativePath := strings.TrimPrefix(path, oldTSDir)
		relativePath = strings.TrimPrefix(relativePath, string(os.PathSeparator))
		if relativePath == "" {
			return nil
		}

		paths.Insert(relativePath)
		return nil
	}

	err := filepath.Walk(oldTSDir, visitor)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	logger.V(5).Info("current paths", "dir", w.targetDir, "paths", sets.List(paths))

	newPaths := sets.New[string]()
	for file := range payload {
		// add all subpaths for the payload to the set of new paths
		// to avoid attempting to remove non-empty dirs
		for subPath := file; subPath != ""; {
			newPaths.Insert(subPath)
			subPath, _ = filepath.Split(subPath)
			subPath = strings.TrimSuffix(subPath, string(os.PathSeparator))
		}
	}
	logger.V(5).Info("new paths", "dir", w.targetDir, "paths", sets.List(newPaths))

	result := paths.Difference(newPaths)
	logger.V(5).Info("paths to remove", "dir", w.targetDir, "paths", result)

	return result, nil
}

// newTimestampDir creates a new timestamp directory
func (w *AtomicWriter) newTimestampDir(logger logr.Logger) (string, error) {
	tsDir, err := os.MkdirTemp(w.targetDir, time.Now().UTC().Format("..2006_01_02_15_04_05."))
	if err != nil {
		logger.Error(err, "unable to create new temp directory")
		return "", err
	}

	// 0755 permissions are needed to allow 'group' and 'other' to recurse the
	// directory tree.  do a chmod here to ensure that permissions are set correctly
	// regardless of the process' umask.
	err = os.Chmod(tsDir, 0755)
	if err != nil {
		logger.Error(err, "unable to set mode on new temp directory")
		return "", err
	}

	return tsDir, nil
}

// writePayloadToDir writes the given payload to the given directory.  The
// directory must exist.
func (w *AtomicWriter) writePayloadToDir(logger logr.Logger, payload map[string]FileProjection, dir string) error {
	for userVisiblePath, fileProjection := range payload {
		content := fileProjection.Data
		mode := os.FileMode(fileProjection.Mode)
		fullPath := filepath.Join(dir, userVisiblePath)
		baseDir, _ := filepath.Split(fullPath)

		if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
			logger.Error(err, "unable to create directory", "dir", baseDir)
			return err
		}

		if err := os.WriteFile(fullPath, content, mode); err != nil {
			logger.Error(err, "unable to write file", "path", fullPath, "mode", mode)
			return err
		}
		// Chmod is needed because os.WriteFile() ends up calling
		// open(2) to create the file, so the final mode used is "mode &
		// ~umask". But we want to make sure the specified mode is used
		// in the file no matter what the umask is.
		if err := os.Chmod(fullPath, mode); err != nil {
			logger.Error(err, "unable to change file mode", "path", fullPath, "mode", mode)
			return err
		}

		if fileProjection.FsUser == nil {
			continue
		}

		uid := ptr.Deref(fileProjection.FsUser, -1)
		gid := ptr.Deref(fileProjection.FsGroup, -1)
		if err := w.chown(logger, fullPath, int(uid), int(gid)); err != nil {
			logger.Error(err, "unable to change file owner", "path", fullPath, "uid", uid, "gid", gid)
			return err
		}
	}

	return nil
}

// createUserVisibleFiles creates the relative symlinks for all the
// files configured in the payload. If the directory in a file path does not
// exist, it is created.
//
// Viz:
// For files: "bar", "foo/bar", "baz/bar", "foo/baz/blah"
// the following symlinks are created:
// bar -> ..data/bar
// foo -> ..data/foo
// baz -> ..data/baz
func (w *AtomicWriter) createUserVisibleFiles(payload map[string]FileProjection) error {
	for userVisiblePath := range payload {
		slashpos := strings.Index(userVisiblePath, string(os.PathSeparator))
		if slashpos == -1 {
			slashpos = len(userVisiblePath)
		}
		linkname := userVisiblePath[:slashpos]
		_, err := os.Readlink(filepath.Join(w.targetDir, linkname))
		if err != nil && os.IsNotExist(err) {
			// The link into the data directory for this path doesn't exist; create it
			visibleFile := filepath.Join(w.targetDir, linkname)
			dataDirFile := filepath.Join(dataDirName, linkname)

			err = os.Symlink(dataDirFile, visibleFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// removeUserVisiblePaths removes the set of paths from the user-visible
// portion of the writer's target directory.
func (w *AtomicWriter) removeUserVisiblePaths(logger logr.Logger, paths sets.Set[string]) error {
	ps := string(os.PathSeparator)
	var lasterr error
	for p := range paths {
		// only remove symlinks from the volume root directory (i.e. items that don't contain '/')
		if strings.Contains(p, ps) {
			continue
		}
		if err := os.Remove(filepath.Join(w.targetDir, p)); err != nil {
			logger.Error(err, "error pruning old user-visible path", "path", p)
			lasterr = err
		}
	}

	return lasterr
}
