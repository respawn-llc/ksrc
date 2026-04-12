package testutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

var (
	zipCentralDirectorySignature = []byte("PK\x01\x02")
	zipEndOfCentralDirSignature  = []byte("PK\x05\x06")
	zipLocalHeaderSignature      = []byte("PK\x03\x04")
)

// CorruptZipEntryMethod rewrites compression method metadata for one entry.
// Useful for tests that need directory metadata intact while body reads fail.
func CorruptZipEntryMethod(path, inner string, method uint16) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	inner = normalizeZipPath(inner)

	centralOffset, err := zipCentralDirectoryOffset(data)
	if err != nil {
		return err
	}

	for pos := centralOffset; pos+46 <= len(data); {
		if !bytes.Equal(data[pos:pos+4], zipCentralDirectorySignature) {
			return fmt.Errorf("zip central directory signature missing at offset %d", pos)
		}

		nameLen := int(binary.LittleEndian.Uint16(data[pos+28 : pos+30]))
		extraLen := int(binary.LittleEndian.Uint16(data[pos+30 : pos+32]))
		commentLen := int(binary.LittleEndian.Uint16(data[pos+32 : pos+34]))
		entryEnd := pos + 46 + nameLen + extraLen + commentLen
		if entryEnd > len(data) {
			return fmt.Errorf("zip central directory entry truncated for %q", inner)
		}

		name := string(data[pos+46 : pos+46+nameLen])
		if normalizeZipPath(name) == inner {
			binary.LittleEndian.PutUint16(data[pos+10:pos+12], method)
			localOffset := int(binary.LittleEndian.Uint32(data[pos+42 : pos+46]))
			if localOffset+30 > len(data) {
				return fmt.Errorf("zip local header truncated for %q", inner)
			}
			if !bytes.Equal(data[localOffset:localOffset+4], zipLocalHeaderSignature) {
				return fmt.Errorf("zip local header signature missing for %q", inner)
			}
			binary.LittleEndian.PutUint16(data[localOffset+8:localOffset+10], method)
			return os.WriteFile(path, data, 0o644)
		}

		pos = entryEnd
	}

	return fmt.Errorf("zip entry not found: %s", inner)
}

func zipCentralDirectoryOffset(data []byte) (int, error) {
	eocd := bytes.LastIndex(data, zipEndOfCentralDirSignature)
	if eocd < 0 || eocd+22 > len(data) {
		return 0, fmt.Errorf("zip end of central directory not found")
	}
	offset := int(binary.LittleEndian.Uint32(data[eocd+16 : eocd+20]))
	if offset < 0 || offset > len(data) {
		return 0, fmt.Errorf("zip central directory offset out of range: %d", offset)
	}
	return offset, nil
}

func normalizeZipPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	return strings.TrimPrefix(path, "/")
}
