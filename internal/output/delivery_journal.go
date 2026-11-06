package output

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const maxPendingDeliveryReceipts = 1_000_000

// deliveryEntry contains no customer event values. An intent is fsynced before
// its append; a completed append is fsynced before its commit receipt. Only an
// input cursor acknowledgement may retire the receipt.
type deliveryEntry struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Path   string `json:"path,omitempty"`
	Offset int64  `json:"offset,omitempty"`
	Length int    `json:"length,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}
type deliveryJournal struct {
	knownSize int64
	path      string
	file      *os.File
	committed map[string]bool
	pending   map[string]deliveryEntry
}

func openDeliveryJournal(path string) (*deliveryJournal, error) {
	if err := makeOwnedDirectory(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	_, priorErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	if os.IsNotExist(priorErr) {
		if err = PreserveFileOwner(filepath.Dir(path), path); err != nil {
			f.Close()
			return nil, err
		}
	}
	if err = syncDeliveryParents(filepath.Dir(path)); err != nil {
		f.Close()
		return nil, err
	}
	j := &deliveryJournal{path: path, file: f, committed: map[string]bool{}, pending: map[string]deliveryEntry{}}
	r := bufio.NewReaderSize(f, 8193)
	var good int64
	for {
		line, err := r.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			f.Close()
			return nil, fmt.Errorf("delivery journal entry exceeds bounds")
		}
		if err == io.EOF {
			if len(line) > 0 {
				if err = f.Truncate(good); err != nil {
					f.Close()
					return nil, err
				}
				if err = f.Sync(); err != nil {
					f.Close()
					return nil, err
				}
			}
			break
		}
		if err != nil {
			f.Close()
			return nil, err
		}
		if len(line) > 8192 {
			f.Close()
			return nil, fmt.Errorf("delivery journal entry exceeds bounds")
		}
		var entry deliveryEntry
		if err = json.Unmarshal(line, &entry); err != nil {
			f.Close()
			return nil, fmt.Errorf("invalid delivery journal: %w", err)
		}
		if len(entry.ID) != 64 {
			f.Close()
			return nil, fmt.Errorf("invalid delivery identity")
		}
		switch entry.Kind {
		case "intent":
			if entry.Length < 1 || entry.Length > 16<<20 || entry.Offset < 0 || entry.Path == "" || len(entry.SHA256) != 64 {
				f.Close()
				return nil, fmt.Errorf("invalid append intent")
			}
			j.pending[entry.ID] = entry
		case "commit":
			j.committed[entry.ID] = true
			delete(j.pending, entry.ID)
		case "abort":
			delete(j.pending, entry.ID)
		default:
			f.Close()
			return nil, fmt.Errorf("invalid delivery journal operation")
		}
		good += int64(len(line))
		if len(j.committed)+len(j.pending) > maxPendingDeliveryReceipts {
			f.Close()
			return nil, fmt.Errorf("unacknowledged local delivery capacity exceeded")
		}
	}
	j.knownSize = good
	return j, nil
}

// MkdirAll may introduce more than one new directory. Syncing only the leaf
// would not persist its entry in the parent after a power interruption.
func syncDeliveryParents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		if err = syncDeliveryDirectory(absolute); err != nil {
			return err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return nil
		}
		absolute = parent
	}
}
func entryComplete(entry deliveryEntry) (bool, error) {
	f, err := os.Open(entry.Path)
	if errors.Is(err, os.ErrNotExist) {
		if entry.Offset == 0 {
			return false, nil
		}
		return false, fmt.Errorf("pending output file disappeared")
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() < entry.Offset {
		return false, fmt.Errorf("pending output was truncated before its append position")
	}
	if info.Size() < entry.Offset+int64(entry.Length) {
		return false, nil
	}
	data := make([]byte, entry.Length)
	if _, err = f.ReadAt(data, entry.Offset); err != nil {
		return false, err
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != entry.SHA256 {
		return false, fmt.Errorf("pending output bytes differ from durable append intent")
	}
	return true, nil
}
func (j *deliveryJournal) append(entry deliveryEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > 8192 {
		return fmt.Errorf("delivery journal entry exceeds bounds")
	}
	if _, err = j.file.Write(data); err != nil {
		return err
	}
	j.knownSize += int64(len(data))
	if err = j.file.Sync(); err != nil {
		j.knownSize = -1
		return err
	}
	return nil
}
func (j *deliveryJournal) close() error {
	if j == nil || j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	return err
}

// WriteIdentified is synchronous for local files: a remote failure can leave an
// input cursor replayable without appending already-durable local rows again.
// Direct Write remains buffered for callers without source identities.
func (w *FileWriter) WriteIdentified(envelope []byte, route, id string) (resultErr error) {
	if !w.acceptsRoute(route) {
		return nil
	}
	if len(id) != 64 {
		return fmt.Errorf("invalid source delivery identity")
	}
	if len(envelope) > 16<<20 {
		return fmt.Errorf("identified output exceeds 16 MiB")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	journalPath, err := w.deliveryPath()
	if err != nil {
		return err
	}
	journalUnlock, err := LockLocalFile(journalPath)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, w.releaseWindowsDelivery(), journalUnlock()) }()
	if err = w.loadDeliveryLocked(journalPath); err != nil {
		return err
	}
	j := w.delivery
	if j.committed[id] {
		return nil
	}
	if len(j.committed)+len(j.pending) >= maxPendingDeliveryReceipts {
		return fmt.Errorf("unacknowledged local delivery capacity exceeded; restore required destination and retry")
	}
	target, err := resolveFilePath(w.cfg, w.now().UTC())
	if err != nil {
		return err
	}
	paths := []string{target, w.currentPath}
	for _, entry := range j.pending {
		paths = append(paths, entry.Path)
	}
	unlock, err := lockLocalPaths(paths...)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, unlock()) }()
	// Release Windows delete-sharing restrictions before queue compaction can
	// acquire this append lock. POSIX writers follow the new inode on next write.
	if runtime.GOOS == "windows" {
		defer func() {
			if w.file != nil {
				resultErr = errors.Join(resultErr, w.file.Close())
				w.file = nil
				w.buffer = nil
			}
		}()
	}
	if err = w.recoverDeliveryLocked(""); err != nil {
		return err
	}
	if j.committed[id] {
		return nil
	}
	path, err := w.ensureCurrentFile()
	if err != nil {
		return err
	}
	maxBytes := w.cfg.MaxMB * 1024 * 1024
	if maxBytes > 0 && w.currentSize+int64(len(envelope))+1 > maxBytes {
		if err = w.rotate(path); err != nil {
			return err
		}
	}
	if err = w.flushAndSyncLocked(); err != nil {
		return err
	}
	info, err := w.file.Stat()
	if err != nil {
		return err
	}
	payload := append(append([]byte(nil), envelope...), '\n')
	hash := sha256.Sum256(payload)
	entry := deliveryEntry{Kind: "intent", ID: id, Path: path, Offset: info.Size(), Length: len(payload), SHA256: hex.EncodeToString(hash[:])}
	if err = j.append(entry); err != nil {
		return err
	}
	j.pending[id] = entry
	if _, err = w.file.Write(payload); err != nil {
		return err
	}
	w.currentSize += int64(len(payload))
	if err = w.file.Sync(); err != nil {
		return err
	}
	if err = syncDeliveryDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	if err = j.append(deliveryEntry{Kind: "commit", ID: id}); err != nil {
		return err
	}
	j.committed[id] = true
	delete(j.pending, id)
	return nil
}
func recoverPartialEntry(entry deliveryEntry) error {
	// No subsequent identified append is admitted while an intent is pending.
	// Therefore a short tail is exclusively this interrupted append. Complete
	// tails were hash-verified at journal load; conflicting full bytes fail safe.
	f, err := os.OpenFile(entry.Path, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() < entry.Offset || info.Size() >= entry.Offset+int64(entry.Length) {
		return fmt.Errorf("cannot recover conflicting local append")
	}
	if err = f.Truncate(entry.Offset); err != nil {
		return err
	}
	return f.Sync()
}
func (w *FileWriter) AcknowledgeDeliveries(ids []string) (resultErr error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := w.deliveryPath()
	if err != nil {
		return err
	}
	if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	unlock, err := LockLocalFile(path)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, w.releaseWindowsDelivery(), unlock()) }()
	if err = w.loadDeliveryLocked(path); err != nil {
		return err
	}
	changed := false
	for _, id := range ids {
		if w.delivery.committed[id] {
			delete(w.delivery.committed, id)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return w.delivery.compact()
}
func (w *FileWriter) AcknowledgeSource(source string, ordinal uint64) (resultErr error) {
	if len(source) != 48 {
		return fmt.Errorf("invalid delivery source identity")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := w.deliveryPath()
	if err != nil {
		return err
	}
	if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	unlock, err := LockLocalFile(path)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, w.releaseWindowsDelivery(), unlock()) }()
	if err = w.loadDeliveryLocked(path); err != nil {
		return err
	}
	changed := false
	for id := range w.delivery.committed {
		if strings.HasPrefix(id, source) {
			position, err := strconv.ParseUint(id[48:], 16, 64)
			if err != nil {
				return fmt.Errorf("invalid delivery ordinal")
			}
			if position <= ordinal {
				delete(w.delivery.committed, id)
				changed = true
			}
		}
	}
	if changed {
		return w.delivery.compact()
	}
	return nil
}
func (j *deliveryJournal) compact() error {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	for id := range j.committed {
		if err := encoder.Encode(deliveryEntry{Kind: "commit", ID: id}); err != nil {
			return err
		}
	}
	for _, entry := range j.pending {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}
	tmp := j.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err = PreserveFileAccess(j.path, tmp); err != nil {
		f.Close()
		return err
	}
	_, err = f.Write(data.Bytes())
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = j.close(); err != nil {
		return err
	}
	renameErr := os.Rename(tmp, j.path)
	j.file, err = os.OpenFile(j.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if renameErr == nil {
		j.knownSize = int64(data.Len())
	}
	return errors.Join(renameErr, err, syncDeliveryDirectory(filepath.Dir(j.path)))
}

func (w *FileWriter) deliveryPath() (string, error) {
	if w.cfg.DeliveryStatePath != "" {
		return w.cfg.DeliveryStatePath, nil
	}
	target, err := resolveFilePath(w.cfg, w.now().UTC())
	return target + ".nli-delivery.wal", err
}
func (w *FileWriter) releaseWindowsDelivery() error {
	if runtime.GOOS == "windows" && w.delivery != nil {
		return w.delivery.close()
	}
	return nil
}

// The caller owns the common WAL lock. Other live writers may have appended
// or atomically compacted the journal since this instance last used it.
func (w *FileWriter) loadDeliveryLocked(path string) error {
	if w.delivery != nil && w.delivery.file != nil {
		opened, err := w.delivery.file.Stat()
		if err != nil {
			return err
		}
		current, err := os.Stat(path)
		if err != nil {
			return err
		}
		if os.SameFile(opened, current) && current.Size() == w.delivery.knownSize {
			return nil
		}
		if err = w.delivery.close(); err != nil {
			return err
		}
	}
	j, err := openDeliveryJournal(path)
	if err != nil {
		return err
	}
	w.delivery = j
	return nil
}
func (w *FileWriter) recoverDeliveryLocked(onlyPath string) error {
	for id, entry := range w.delivery.pending {
		if onlyPath != "" {
			a, _ := filepath.Abs(onlyPath)
			b, _ := filepath.Abs(entry.Path)
			if a != b {
				continue
			}
		}
		complete, err := entryComplete(entry)
		if err != nil {
			return err
		}
		kind := "abort"
		if complete {
			kind = "commit"
			w.delivery.committed[id] = true
		} else {
			if err = recoverPartialEntry(entry); err != nil {
				return err
			}
			if w.currentPath == entry.Path {
				w.currentSize = entry.Offset
			}
		}
		if err = w.delivery.append(deliveryEntry{Kind: kind, ID: id}); err != nil {
			return err
		}
		delete(w.delivery.pending, id)
	}
	return nil
}

// RecoverLocalDeliveryLocked is used only while LockLocalDelivery owns this
// destination's WAL+append locks, before replay snapshots or replaces its queue.
func RecoverLocalDeliveryLocked(cfg Config, path string) error {
	if cfg.DeliveryStatePath == "" {
		return fmt.Errorf("delivery journal path is required")
	}
	if _, err := os.Stat(cfg.DeliveryStatePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	w := &FileWriter{cfg: cfg}
	if err := w.loadDeliveryLocked(cfg.DeliveryStatePath); err != nil {
		return err
	}
	defer w.delivery.close()
	return w.recoverDeliveryLocked(path)
}
