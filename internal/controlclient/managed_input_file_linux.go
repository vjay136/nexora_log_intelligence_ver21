package controlclient

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
	"nexora.local/nli-v20/internal/agent"
	"nexora.local/nli-v20/internal/protocol"
)

const managedInputCopyLimit = 64 << 20

func openManagedInputDirectory(path string) (int, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return -1, fmt.Errorf("invalid managed input directory")
	}
	fd, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		child, err := unix.Openat(fd, part, unix.O_DIRECTORY|unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if err != nil {
			return -1, err
		}
		fd = child
	}
	return fd, nil
}

func listManagedInputFiles(root string) ([]string, bool) {
	fd, err := openManagedInputDirectory(root)
	if err != nil {
		return nil, false
	}
	directory := os.NewFile(uintptr(fd), root)
	defer directory.Close()
	names, err := directory.Readdirnames(257)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false
	}
	files := []string{}
	for _, name := range names {
		if len(files) == 256 {
			break
		}
		if (protocol.ManagedInputFileRequest{Action: "delete", Source: name}).Validate() != nil {
			continue
		}
		var info unix.Stat_t
		if unix.Fstatat(fd, name, &info, unix.AT_SYMLINK_NOFOLLOW) == nil && info.Mode&unix.S_IFMT == unix.S_IFREG && info.Nlink == 1 {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, true
}

func checkUnconfiguredManagedInput(configPath, path string) error {
	if configPath == path {
		return fmt.Errorf("managed filename refers to the agent configuration")
	}
	if resolved, err := filepath.EvalSymlinks(configPath); err == nil && resolved == path {
		return fmt.Errorf("managed filename refers to the agent configuration")
	}
	config, err := agent.Load(configPath)
	if err != nil {
		return fmt.Errorf("cannot verify active input configuration: %w", err)
	}
	for _, input := range config.Inputs {
		pattern := filepath.Clean(input.Path)
		if strings.ContainsAny(pattern, "*?[") {
			parent := filepath.Dir(pattern)
			if strings.ContainsAny(parent, "*?[") {
				return fmt.Errorf("cannot verify input coverage while directory wildcard inputs are configured")
			}
			if resolved, err := filepath.EvalSymlinks(parent); err == nil {
				pattern = filepath.Join(resolved, filepath.Base(pattern))
			}
			matches, err := filepath.Match(pattern, path)
			if err != nil || matches {
				return fmt.Errorf("managed filename is covered by a configured input pattern")
			}
			continue
		}
		if pattern == path {
			return fmt.Errorf("remove the file from collection and verify the input rollout first")
		}
		if resolved, err := filepath.EvalSymlinks(pattern); err == nil && resolved == path {
			return fmt.Errorf("managed filename is used by a configured input alias")
		}
	}
	return nil
}

func manageInputFile(root, configPath string, request protocol.ManagedInputFileRequest) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}
	target := request.Source
	if request.Action == "copy" {
		target = request.Destination
	}
	if err := checkUnconfiguredManagedInput(configPath, filepath.Join(root, target)); err != nil {
		return "", err
	}
	fd, err := openManagedInputDirectory(root)
	if err != nil {
		return "", fmt.Errorf("open managed input directory: %w", err)
	}
	defer unix.Close(fd)
	var info unix.Stat_t
	if err := unix.Fstatat(fd, request.Source, &info, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if request.Action == "delete" && errors.Is(err, unix.ENOENT) {
			return "managed input file already absent", nil
		}
		return "", fmt.Errorf("inspect managed input file: %w", err)
	}
	if info.Mode&unix.S_IFMT != unix.S_IFREG || info.Nlink != 1 {
		return "", fmt.Errorf("managed input operations require a regular file without hard links")
	}
	if request.Action == "delete" {
		if err := unix.Unlinkat(fd, request.Source, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			return "", fmt.Errorf("delete managed input file: %w", err)
		}
		if err := unix.Fsync(fd); err != nil {
			return "", fmt.Errorf("file deleted but directory sync failed: %w", err)
		}
		return "managed input file deleted", nil
	}
	if err := copyManagedInputFile(fd, request, info); err != nil {
		return "", err
	}
	return "managed input file copied", nil
}

func copyManagedInputFile(dir int, request protocol.ManagedInputFileRequest, expected unix.Stat_t) error {
	sourceFD, err := unix.Openat(dir, request.Source, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open managed input source: %w", err)
	}
	source := os.NewFile(uintptr(sourceFD), request.Source)
	defer source.Close()
	var actual unix.Stat_t
	if err := unix.Fstat(sourceFD, &actual); err != nil || actual.Ino != expected.Ino || actual.Dev != expected.Dev || actual.Mode != expected.Mode || actual.Nlink != 1 {
		return fmt.Errorf("managed input source changed before copying")
	}
	if actual.Size > managedInputCopyLimit {
		return fmt.Errorf("managed input copy exceeds the 64 MiB limit")
	}
	tempFD, err := unix.Openat(dir, ".", unix.O_TMPFILE|unix.O_RDWR|unix.O_CLOEXEC, 0600)
	if err != nil {
		return fmt.Errorf("stage managed input copy: %w", err)
	}
	temp := os.NewFile(uintptr(tempFD), "managed input copy")
	defer temp.Close()
	count, err := io.Copy(temp, io.LimitReader(source, managedInputCopyLimit+1))
	if err != nil {
		return fmt.Errorf("copy managed input file: %w", err)
	}
	if err := unix.Fstat(sourceFD, &actual); err != nil || count != expected.Size || actual.Size != expected.Size || actual.Mtim != expected.Mtim || actual.Ctim != expected.Ctim {
		return fmt.Errorf("managed input source changed during copying; retry when it is idle")
	}
	if err := unix.Fchown(tempFD, int(actual.Uid), int(actual.Gid)); err != nil {
		return fmt.Errorf("preserve managed input file ownership: %w", err)
	}
	if err := unix.Fchmod(tempFD, actual.Mode&0640); err != nil {
		return fmt.Errorf("preserve managed input file permissions: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync managed input copy: %w", err)
	}
	err = unix.Linkat(tempFD, "", dir, request.Destination, unix.AT_EMPTY_PATH)
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.EPERM) {
		// Linux may restrict AT_EMPTY_PATH to processes with DAC_READ_SEARCH.
		err = unix.Linkat(unix.AT_FDCWD, fmt.Sprintf("/proc/self/fd/%d", tempFD), dir, request.Destination, unix.AT_SYMLINK_FOLLOW)
	}
	if err != nil {
		if errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("copy destination already exists; choose a new name")
		}
		return fmt.Errorf("publish managed input copy: %w", err)
	}
	if err := unix.Fsync(dir); err != nil {
		return fmt.Errorf("file copied but directory sync failed: %w", err)
	}
	return nil
}
