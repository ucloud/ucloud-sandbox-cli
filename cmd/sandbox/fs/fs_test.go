package fs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCommandsAreRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command().Commands() {
		names[c.Name()] = true
	}

	for _, name := range []string{"cat", "cp", "ls", "mkdir", "mv", "rm"} {
		assert.True(t, names[name], "fs %s is missing", name)
	}
}

// Every fs command needs a user: envd resolves relative paths and permissions
// against one.
func TestEveryCommandTakesAUser(t *testing.T) {
	operations := map[string]cmd.Operation{
		"cat":   &catOperation{},
		"cp":    &cpOperation{},
		"ls":    &lsOperation{},
		"mkdir": &mkdirOperation{},
		"mv":    &mvOperation{},
		"rm":    &rmOperation{},
	}

	for name, op := range operations {
		t.Run(name, func(t *testing.T) {
			c := op.Command()

			flag := c.Flags().Lookup("user")
			require.NotNil(t, flag, "fs %s has no --user", name)
			assert.Equal(t, "u", flag.Shorthand)

			require.NoError(t, c.ParseFlags([]string{"-u", "root"}))
		})
	}
}

func TestCatFlags(t *testing.T) {
	op := &catOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-u", "root"}))

	assert.Equal(t, "root", op.user)
	assert.Empty(t, (&catOperation{}).user, "the user is empty until the flag is given")
}

func TestLsFlags(t *testing.T) {
	op := &lsOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags([]string{"-u", "root", "-d", "3", "-f", "json", "-p", "2", "-l", "10"}))

	assert.Equal(t, "root", op.user)
	assert.Equal(t, uint32(3), op.depth)
	assert.Equal(t, list.FormatJSON, op.list.Format)
	assert.Equal(t, 2, op.list.Page)
	assert.Equal(t, 10, op.list.Limit)
}

func TestLsDefaults(t *testing.T) {
	op := &lsOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(nil))

	assert.Equal(t, uint32(1), op.depth, "a listing stops at the directory's own entries")
	assert.Equal(t, list.FormatPretty, op.list.Format)
	assert.NoError(t, op.list.Validate())
}

func TestLsArgCount(t *testing.T) {
	c := (&lsOperation{}).Command()

	assert.Error(t, c.Args(c, nil), "the sandbox is required")
	assert.NoError(t, c.Args(c, []string{"sbx-1"}), "the path defaults to the working directory")
	assert.NoError(t, c.Args(c, []string{"sbx-1", "/home/user"}))
	assert.Error(t, c.Args(c, []string{"sbx-1", "/home/user", "extra"}))
}

func TestEntryType(t *testing.T) {
	assert.Equal(t, "file", entryType(filesystem.FileType_FILE_TYPE_FILE))
	assert.Equal(t, "dir", entryType(filesystem.FileType_FILE_TYPE_DIRECTORY))
	assert.Equal(t, "symlink", entryType(filesystem.FileType_FILE_TYPE_SYMLINK))
	assert.Equal(t, "unknown", entryType(filesystem.FileType_FILE_TYPE_UNSPECIFIED))
}

func TestToListedEntry(t *testing.T) {
	modified := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.UTC)
	target := "/etc/alternatives/vim"

	row := toListedEntry(&filesystem.EntryInfo{
		Name:          "vim",
		Path:          "/usr/bin/vim",
		Type:          filesystem.FileType_FILE_TYPE_SYMLINK,
		Size:          1234,
		Permissions:   "lrwxrwxrwx",
		Owner:         "root",
		Group:         "root",
		ModifiedTime:  timestamppb.New(modified),
		SymlinkTarget: &target,
	})

	assert.Equal(t, listedEntry{
		Name:          "vim",
		Path:          "/usr/bin/vim",
		Type:          "symlink",
		Size:          1234,
		Permissions:   "lrwxrwxrwx",
		Owner:         "root",
		Group:         "root",
		ModifiedTime:  modified,
		SymlinkTarget: target,
	}, row)
}

func TestToListedEntryWithoutOptionalFields(t *testing.T) {
	row := toListedEntry(&filesystem.EntryInfo{Name: "notes.txt", Type: filesystem.FileType_FILE_TYPE_FILE})

	assert.Equal(t, "notes.txt", row.Name)
	assert.Equal(t, "file", row.Type)
	assert.True(t, row.ModifiedTime.IsZero(), "an entry with no timestamp reports none")
	assert.Empty(t, row.SymlinkTarget)
}

func TestToListedEntryOfNil(t *testing.T) {
	// The proto getters are nil-safe, so a missing entry renders as an empty
	// row rather than panicking.
	assert.Equal(t, listedEntry{Type: "unknown"}, toListedEntry(nil))
}
