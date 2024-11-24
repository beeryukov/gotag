// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
)

// blockType is a type which represents an enumeration of valid FLAC blocks
type blockType byte

// FLAC block types.
const (
	// Stream Info Block           0
	// Padding Block               1
	// Application Block           2
	// Seektable Block             3
	// Cue Sheet Block             5
	vorbisCommentBlock blockType = 4
	pictureBlock       blockType = 6
)

// ReadFLACTags reads FLAC metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
func ReadFLACTags(r io.ReadSeeker) (Metadata, error) {
	flac, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if flac != "fLaC" {
		return nil, errors.New("expected 'fLaC'")
	}

	m := &metadataFLAC{
		newMetadataVorbis(),
	}

	for {
		last, err := m.readFLACMetadataBlock(r)
		if err != nil {
			return nil, err
		}

		if last {
			break
		}
	}
	return m, nil
}

func WriteFLACTags(rw io.ReadWriteSeeker, data map[string]string) error {
	flac, err := readString(rw, 4)
	if err != nil {
		return err
	}
	if flac != "fLaC" {
		return errors.New("expected 'fLaC'")
	}

	return findAndWriteFlacCommentBlock(rw, data)
}

type metadataFLAC struct {
	*metadataVorbis
}

func findAndWriteFlacCommentBlock(rw io.ReadWriteSeeker, data map[string]string) error {

	for {
		blockHeader, err := readBytes(rw, 1)
		if err != nil {
			return err
		}

		if getBit(blockHeader[0], 7) {
			blockHeader[0] ^= (1 << 7)
			break
		}

		blockLen, err := readInt(rw, 3)
		if err != nil {
			return err
		}

		switch blockType(blockHeader[0]) {
		case vorbisCommentBlock:

			preparedVorbisComment := PrepareVorbisComment(data)

			newBlockLen := len(preparedVorbisComment)
			blockLenBytes := formatIntBigEndian(uint(newBlockLen), 3)
			rw.Seek(-3, io.SeekCurrent)
			rw.Write(blockLenBytes)

			if newBlockLen <= blockLen {
				n, err := rw.Write(preparedVorbisComment)
				if err != nil {
					return err
				}
				if n < len(preparedVorbisComment) {
					return errors.New("number of bytes written to file is less than tags length")
				}

				shift := blockLen - newBlockLen
				if shift > 0 {
					return ShiftFileLeft(rw, shift)
				}
			} else {
				blockDataStartPos, _ := rw.Seek(0, io.SeekCurrent)
				rw.Seek(int64(blockLen), io.SeekCurrent)
				ShiftFileRight(rw, newBlockLen-blockLen)
				rw.Seek(blockDataStartPos, io.SeekStart)

				n, err := rw.Write(preparedVorbisComment)
				if err != nil {
					return err
				}
				if n < len(preparedVorbisComment) {
					return errors.New("number of bytes written to file is less than tags length")
				}
				return nil
			}

		default:
			_, err = rw.Seek(int64(blockLen), io.SeekCurrent)
		}
	}
	return nil
}

// ShiftFileLeft На момент вызова функции, rw должен быть в позиции,
// к которой подтянется содержимое файла, находящееся на offset байт правее этой позиции
func ShiftFileLeft(rw io.ReadWriteSeeker, offset int) error {
	originalPosition, _ := rw.Seek(0, io.SeekCurrent)

	buf := make([]byte, 1024*1024)

	for {
		_, err := rw.Seek(int64(offset), io.SeekCurrent)
		if err != nil {
			return err
		}
		n, err := rw.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n > 0 {
			_, err := rw.Seek(-int64(n+offset), io.SeekCurrent)
			if err != nil {
				return err
			}
			_, err = rw.Write(buf[:n])

			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}
	}
	rw.Seek(originalPosition, io.SeekStart)

	return nil
}

// ShiftFileRight На момент вызова функции, rw должен быть в позиции,
// от которой всё дальнейшее содержимое файла будет отодвинуто на offset байт
func ShiftFileRight(rw io.ReadWriteSeeker, offset int) {
	originalPosition, _ := rw.Seek(0, io.SeekCurrent)
	fileSize, _ := rw.Seek(0, io.SeekEnd)
	rw.Seek(originalPosition, io.SeekStart)

	bufSize := 100
	stop := false

	if int(fileSize-originalPosition) < bufSize {
		bufSize = int(fileSize - originalPosition)
		stop = true
	}
	buf := make([]byte, bufSize)
	bufLen64 := int64(len(buf))

	rw.Seek(-bufLen64, io.SeekEnd)

	for {
		rw.Read(buf)
		rw.Seek(-bufLen64+int64(offset), io.SeekCurrent)
		rw.Write(buf)
		if stop {
			rw.Seek(-bufLen64-int64(offset), io.SeekCurrent)
			buf = buf[:offset]
			for i := range buf {
				buf[i] = []byte("_")[0]
			}
			rw.Write(buf)
			break
		}
		curPos, _ := rw.Seek(-bufLen64-int64(offset), io.SeekCurrent)

		if curPos-originalPosition >= bufLen64 {
			rw.Seek(-bufLen64, io.SeekCurrent)
		} else {
			rw.Seek(-(curPos - originalPosition), io.SeekCurrent)
			buf = buf[:(curPos - originalPosition)]
			bufLen64 = int64(len(buf))
			stop = true
		}
	}

	rw.Seek(originalPosition, io.SeekStart)
}

func (m *metadataFLAC) readFLACMetadataBlock(r io.ReadSeeker) (last bool, err error) {
	blockHeader, err := readBytes(r, 1)
	if err != nil {
		return
	}

	if getBit(blockHeader[0], 7) {
		blockHeader[0] ^= (1 << 7)
		last = true
	}

	blockLen, err := readInt(r, 3)
	if err != nil {
		return
	}

	switch blockType(blockHeader[0]) {
	case vorbisCommentBlock:
		err = m.readVorbisComment(r)

	case pictureBlock:
		err = m.readPictureBlock(r)

	default:
		_, err = r.Seek(int64(blockLen), io.SeekCurrent)
	}
	return
}

func (m *metadataFLAC) FileType() FileType {
	return FLAC
}
