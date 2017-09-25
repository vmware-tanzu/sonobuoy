package tarx

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompressFile(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/a.txt", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	headers, err := List(filename)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(headers))
	assert.Equal(t, "a.txt", headers[0].Name)
}

func TestCompressFolder(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	headers, err := List(filename)
	assert.NoError(t, err)

	assert.Equal(t, 6, len(headers))
	assert.Equal(t, "a.txt", headers[0].Name)
	assert.Equal(t, "b.txt", headers[1].Name)
	assert.Equal(t, "c", headers[2].Name)
	assert.Equal(t, "c/c1.txt", headers[3].Name)
	assert.Equal(t, "c/c2.txt", headers[4].Name)
	assert.Equal(t, "symlink.txt", headers[5].Name)
}

func TestCompressFolderWithIncludeSourceDir(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", &CompressOptions{IncludeSourceDir: true})
	assert.NoError(t, err)
	defer os.Remove(filename)

	headers, err := List(filename)
	assert.NoError(t, err)

	assert.Equal(t, 7, len(headers))
	assert.Equal(t, "input", headers[0].Name)
	assert.Equal(t, "input/a.txt", headers[1].Name)
	assert.Equal(t, "input/b.txt", headers[2].Name)
	assert.Equal(t, "input/c", headers[3].Name)
	assert.Equal(t, "input/c/c1.txt", headers[4].Name)
	assert.Equal(t, "input/c/c2.txt", headers[5].Name)
	assert.Equal(t, "input/symlink.txt", headers[6].Name)
}

func TestAppendFile(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/c", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	err = Compress(filename, "tests/input/a.txt", &CompressOptions{Append: true})
	assert.NoError(t, err)

	headers, err := List(filename)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(headers))
	assert.Equal(t, "c1.txt", headers[0].Name)
	assert.Equal(t, "c2.txt", headers[1].Name)
	assert.Equal(t, "a.txt", headers[2].Name)
}

func TestAppendFileWithGzip(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/c", &CompressOptions{Compression: Gzip})
	assert.NoError(t, err)
	defer os.Remove(filename)

	err = Compress(filename, "tests/input/a.txt", &CompressOptions{Append: true})
	assert.EqualError(t, ErrAppendNotSupported, err.Error())
}

func TestFindFile(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/a.txt", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	header, reader, err := Find(filename, "a.txt")
	assert.Equal(t, nil, err)
	assert.Equal(t, "a.txt", header.Name)
	b, _ := ioutil.ReadAll(reader)
	assert.Equal(t, "a.txt\n", string(b))
	assert.Equal(t, nil, reader.Close())
}

func TestFindDir(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	_, reader, err := Find(filename, "c")
	assert.Equal(t, nil, reader)
	assert.Equal(t, nil, err)
}

func TestReadNotExists(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/a.txt", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	_, _, err = Find(filename, "notExists.txt")
	assert.Equal(t, os.ErrNotExist, err)
}

func TestExtract(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	err = Extract(filename, "tests/output", nil)
	assert.NoError(t, err)
	defer os.RemoveAll("tests/output")

	assert.Equal(t, true, pathExists("tests/output/a.txt"))
	assert.Equal(t, true, pathExists("tests/output/b.txt"))
	assert.Equal(t, true, pathExists("tests/output/symlink.txt"))
	assert.Equal(t, true, pathExists("tests/output/c"))
	assert.Equal(t, true, pathExists("tests/output/c/c1.txt"))
	assert.Equal(t, true, pathExists("tests/output/c/c2.txt"))
}

func TestExtractWithFlatDir(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	err = Extract(filename, "tests/output", &ExtractOptions{FlatDir: true})
	assert.NoError(t, err)
	defer os.RemoveAll("tests/output")

	assert.Equal(t, true, pathExists("tests/output/a.txt"))
	assert.Equal(t, true, pathExists("tests/output/b.txt"))
	assert.Equal(t, true, pathExists("tests/output/symlink.txt"))
	assert.Equal(t, false, pathExists("tests/output/c"))
	assert.Equal(t, true, pathExists("tests/output/c1.txt"))
	assert.Equal(t, true, pathExists("tests/output/c2.txt"))
}

func TestExtractWithFilters(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	filters := []string{"a.txt", "c/c2.txt"}
	err = Extract(filename, "tests/output", &ExtractOptions{Filters: filters})
	assert.NoError(t, err)
	defer os.RemoveAll("tests/output")

	assert.Equal(t, true, pathExists("tests/output/a.txt"))
	assert.Equal(t, false, pathExists("tests/output/b.txt"))
	assert.Equal(t, false, pathExists("tests/output/symlink.txt"))
	assert.Equal(t, true, pathExists("tests/output/c"))
	assert.Equal(t, false, pathExists("tests/output/c/c1.txt"))
	assert.Equal(t, true, pathExists("tests/output/c/c2.txt"))
}

func TestExtractWithOverride(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	os.MkdirAll("tests/output/c", os.ModePerm)
	writeContent("tests/output/a.txt", "new a.txt")
	writeContent("tests/output/c/z.txt", "z.txt")

	err = Extract(filename, "tests/output", nil)
	assert.NoError(t, err)
	defer os.RemoveAll("tests/output")

	assert.Equal(t, true, pathExists("tests/output/a.txt"))
	assert.Equal(t, true, pathExists("tests/output/b.txt"))
	assert.Equal(t, true, pathExists("tests/output/c"))
	assert.Equal(t, true, pathExists("tests/output/c/c1.txt"))
	assert.Equal(t, true, pathExists("tests/output/c/c2.txt"))
	assert.Equal(t, true, pathExists("tests/output/c/z.txt"))

	assert.Equal(t, "a.txt\n", readContent("tests/output/a.txt"))
	assert.Equal(t, "z.txt", readContent("tests/output/c/z.txt"))
}

func TestExtractWithoutOverride(t *testing.T) {
	filename := "tests/test.tar"

	err := Compress(filename, "tests/input/a.txt", nil)
	assert.NoError(t, err)
	defer os.Remove(filename)

	os.MkdirAll("tests/output", os.ModePerm)
	writeContent("tests/output/a.txt", "new a.txt")

	err = Extract(filename, "tests/output", &ExtractOptions{NoOverride: true})
	assert.NoError(t, err)
	defer os.RemoveAll("tests/output")

	assert.Equal(t, true, pathExists("tests/output/a.txt"))
	assert.Equal(t, "new a.txt", readContent("tests/output/a.txt"))
}

func pathExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		return false
	}
	return true
}

func readContent(filePath string) string {
	file, _ := os.OpenFile(filePath, os.O_RDWR, os.ModePerm)
	defer file.Close()
	content, _ := ioutil.ReadAll(file)
	return string(content)
}

func writeContent(filePath, content string) {
	file, _ := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	file.WriteString(content)
}
