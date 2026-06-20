package banning

import (
	"testing"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func infoWithName(name string) metainfo.Info {
	return metainfo.Info{
		Name:   name,
		Length: 65536,
		Pieces: make([]byte, 20),
	}
}

func infoWithSize(size int64) metainfo.Info {
	return metainfo.Info{
		Name:   "testfile.txt",
		Length: size,
		Pieces: make([]byte, 20),
	}
}

func infoWithFiles(name string, files []metainfo.FileInfo) metainfo.Info {
	return metainfo.Info{
		Name:   name,
		Files:  files,
		Pieces: make([]byte, 20),
	}
}

func TestNameLengthChecker_TooShort(t *testing.T) {
	t.Parallel()

	c := nameLengthChecker{min: 8}

	err := c.Check(infoWithName("short"))
	require.ErrorContains(t, err, "name too short")

	err = c.Check(infoWithName("longenough"))
	assert.NoError(t, err)
}

func TestNameLengthChecker_EmptyName(t *testing.T) {
	t.Parallel()

	c := nameLengthChecker{min: 1}

	err := c.Check(infoWithName(""))
	assert.ErrorContains(t, err, "name too short")
}

func TestNameLengthChecker_UsesBestName(t *testing.T) {
	t.Parallel()

	c := nameLengthChecker{min: 8}

	info := metainfo.Info{
		Name:     "short",
		NameUtf8: "longenough",
		Length:   65536,
		Pieces:   make([]byte, 20),
	}

	err := c.Check(info)
	assert.NoError(t, err)
}

func TestSizeChecker_TooSmall(t *testing.T) {
	t.Parallel()

	c := sizeChecker{min: 1024}

	err := c.Check(infoWithSize(100))
	require.ErrorContains(t, err, "size too small")

	err = c.Check(infoWithSize(2048))
	assert.NoError(t, err)
}

func TestSizeChecker_ExactMinimum(t *testing.T) {
	t.Parallel()

	c := sizeChecker{min: 1024}

	err := c.Check(infoWithSize(1024))
	assert.NoError(t, err)
}

func TestSizeChecker_MultiFileTotal(t *testing.T) {
	t.Parallel()

	c := sizeChecker{min: 1000}

	info := metainfo.Info{
		Name: "dir",
		Files: []metainfo.FileInfo{
			{Length: 600, Path: []string{"a.txt"}},
			{Length: 500, Path: []string{"b.txt"}},
		},
		Pieces: make([]byte, 20),
	}

	err := c.Check(info)
	require.NoError(t, err)

	infoSmall := metainfo.Info{
		Name: "dir",
		Files: []metainfo.FileInfo{
			{Length: 100, Path: []string{"a.txt"}},
		},
		Pieces: make([]byte, 20),
	}

	err = c.Check(infoSmall)
	assert.ErrorContains(t, err, "size too small")
}

func TestUtf8Checker_Valid(t *testing.T) {
	t.Parallel()

	var c utf8Checker

	err := c.Check(infoWithName("hello.txt"))
	require.NoError(t, err)

	err = c.Check(infoWithName("\u65e5\u672c\u8a9e.txt"))
	require.NoError(t, err)

	err = c.Check(infoWithName("résumé.pdf"))
	assert.NoError(t, err)
}

func TestUtf8Checker_Invalid(t *testing.T) {
	t.Parallel()

	var c utf8Checker

	info := metainfo.Info{
		Name:   "hello\xff.txt",
		Length: 65536,
		Pieces: make([]byte, 20),
	}

	err := c.Check(info)
	assert.ErrorContains(t, err, "invalid utf8")
}

func TestUtf8Checker_NullByte(t *testing.T) {
	t.Parallel()

	var c utf8Checker

	info := metainfo.Info{
		Name:   "hello\x00.txt",
		Length: 65536,
		Pieces: make([]byte, 20),
	}

	err := c.Check(info)
	assert.ErrorContains(t, err, "invalid utf8")
}

func TestUtf8Checker_ChecksFiles(t *testing.T) {
	t.Parallel()

	var c utf8Checker

	info := metainfo.Info{
		Name: "dir",
		Files: []metainfo.FileInfo{
			{Length: 100, Path: []string{"good.txt"}},
			{Length: 200, Path: []string{"bad\xff.dat"}},
		},
		Pieces: make([]byte, 20),
	}

	err := c.Check(info)
	assert.ErrorContains(t, err, "invalid utf8")
}

func TestContainsGarbage_ReplacementChar(t *testing.T) {
	t.Parallel()

	assert.True(t, containsGarbage("hello\uFFFDworld"))
	assert.False(t, containsGarbage("hello world"))
}

func TestContainsGarbage_ControlChars(t *testing.T) {
	t.Parallel()

	assert.True(t, containsGarbage("hello\x00world"))
	assert.True(t, containsGarbage("hello\x1Bworld"))
	assert.False(t, containsGarbage("hello\tworld"))
	assert.False(t, containsGarbage("hello\nworld"))
	assert.False(t, containsGarbage("hello\rworld"))
}

func TestContainsGarbage_PUA(t *testing.T) {
	t.Parallel()

	assert.True(t, containsGarbage("hello\uE000world"))
	assert.True(t, containsGarbage(string(rune(0xF0000))))
}

func TestContainsGarbage_MixedScripts(t *testing.T) {
	t.Parallel()

	assert.True(t, containsGarbage("\u65e5\u672c\u8a9eabc\uFF76"))
	assert.False(t, containsGarbage("\u65e5\u672c\u8a9eabc"))
	assert.False(t, containsGarbage("\u65e5\u672c\u8a9e\uFF76"))
}

func TestContainsGarbage_Clean(t *testing.T) {
	t.Parallel()

	assert.False(t, containsGarbage("hello world"))
	assert.False(t, containsGarbage("test_file.txt"))
	assert.False(t, containsGarbage("12345"))
	assert.False(t, containsGarbage(""))
}

func TestContentChecker_Valid(t *testing.T) {
	t.Parallel()

	var c contentChecker

	err := c.Check(infoWithName("valid_name.txt"))
	assert.NoError(t, err)
}

func TestContentChecker_GarbageInName(t *testing.T) {
	t.Parallel()

	var c contentChecker

	err := c.Check(infoWithName("bad\x00name.txt"))
	assert.ErrorContains(t, err, "garbage")
}

func TestContentChecker_GarbageInFile(t *testing.T) {
	t.Parallel()

	var c contentChecker

	info := infoWithFiles("dir", []metainfo.FileInfo{
		{Length: 100, Path: []string{"good.txt"}},
		{Length: 200, Path: []string{"bad\uFFFdfile.dat"}},
	})

	err := c.Check(info)
	assert.ErrorContains(t, err, "garbage")
}

func TestCombinedChecker_AllPass(t *testing.T) {
	t.Parallel()

	c := combinedChecker{
		checkers: []Checker{
			nameLengthChecker{min: 1},
			sizeChecker{min: 1},
		},
	}

	err := c.Check(infoWithName("valid.txt"))
	assert.NoError(t, err)
}

func TestCombinedChecker_SomeFail(t *testing.T) {
	t.Parallel()

	c := combinedChecker{
		checkers: []Checker{
			nameLengthChecker{min: 100},
			sizeChecker{min: 1024 * 1024},
		},
	}

	err := c.Check(infoWithName("short"))
	require.Error(t, err)
	require.ErrorContains(t, err, "name too short")
	require.ErrorContains(t, err, "size too small")

	c2 := combinedChecker{
		checkers: []Checker{
			sizeChecker{min: 65536 * 2},
		},
	}

	err = c2.Check(infoWithName("long_enough_name"))
	assert.ErrorContains(t, err, "size too small")
}

func TestNew_SetsDefaultCheckers(t *testing.T) {
	t.Parallel()

	p := Params{}
	result := New(p)
	assert.NotNil(t, result.Checker)

	info := metainfo.Info{
		Name:   "hi",
		Length: 10,
		Pieces: make([]byte, 20),
	}

	err := result.Checker.Check(info)
	assert.Error(t, err)
}
