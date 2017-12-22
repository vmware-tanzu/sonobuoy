package tarball

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

func DecodeTarball(reader io.Reader, baseDir string) error {
	gzStream, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Wrap(err, "couldn't uncompress reader")
	}

	tarchive := tar.NewReader(gzStream)
	for {
		header, err := tarchive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "couldn't opening tarball from gzip")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path.Join(baseDir, header.Name), os.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
		case tar.TypeReg, tar.TypeRegA:
			filePath := path.Join(baseDir, header.Name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
			file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, "error decoding tarball for result (open)")
			}
			if _, err := io.CopyN(file, tarchive, header.Size); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (copy)")
			}
		case tar.TypeSymlink:
			filePath := path.Join(baseDir, header.Name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
				return errors.Wrapf(err, "error decoding tarball for result (mkdir)")
			}
			if err := os.Symlink(path.Join(baseDir, header.Linkname), path.Join(baseDir, header.Name)); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (ln)")
			}
		default:
		}
	}

	return nil
}
