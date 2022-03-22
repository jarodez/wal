package log_test

import (
	"io/ioutil"
	"os"
	"testing"

	api "github.com/intelitecs/wal/api/v1/log"
	wal_log "github.com/intelitecs/wal/internal/log"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T, log *wal_log.Log,
	){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOffsetOutOfRange,
		//"init with existing segments":       testInitWithSegments,
		"reader":   testReader,
		"truncate": testTruncate,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "store-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			cfg := wal_log.Config{}
			cfg.Segment.MaxStoreBytes = 32
			log, err := wal_log.NewLog(dir, cfg)
			require.NoError(t, err)
			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *wal_log.Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	off, err := log.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, append.Value, read.Value)
}

func testInitWithSegments(t *testing.T, log *wal_log.Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	for i := 0; i < 3; i++ {
		_, err := log.Append(append)
		require.NoError(t, err)
	}
	require.NoError(t, log.Close())

	off, err := log.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = log.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)
	n, err := wal_log.NewLog(log.Dir, log.Config)
	require.NoError(t, err)
	off, err = n.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = n.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)
}

func testReader(t *testing.T, log *wal_log.Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	off, err := log.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	reader := log.Reader()
	b, err := ioutil.ReadAll(reader)
	require.NoError(t, err)
	read := &api.Record{}
	err = proto.Unmarshal(b[wal_log.LenWidth:], read)
	require.NoError(t, err)
	require.Equal(t, append.Value, read.Value)
}

func testTruncate(t *testing.T, log *wal_log.Log) {
	append := &api.Record{Value: []byte("hello world")}
	for i := 0; i < 3; i++ {
		_, err := log.Append(append)
		require.NoError(t, err)
	}

	err := log.Truncate(1)
	require.NoError(t, err)
	_, err = log.Read(0)
	require.NoError(t, err)
}

func testOffsetOutOfRange(t *testing.T, log *wal_log.Log) {
	read, err := log.Read(1)
	require.Nil(t, read)
	apiErr := err.(wal_log.ErrOffsetOutOfRange)
	require.Equal(t, uint64(1), apiErr.Offset)
}