package cmd

import (
	"os"
	"testing"
)
import "github.com/stretchr/testify/assert"

func TestSegments(t *testing.T) {

	seg := getSegments("")
	assert.Nil(t, seg)

	seg = getSegments("test")
	assert.Equal(t, 1, len(seg))

	assert.True(t, segmentMatch("2023/games/game12:t1:test:t2", seg))
	assert.False(t, segmentMatch("2023/games/game12:t1:tes:t2", seg))
	assert.False(t, segmentMatch("test/test/game12:t1:tes:t2", seg))

	seg = getSegments("test:t2")
	assert.True(t, segmentMatch("2023/games/game12:t1:test:t2", seg))
	assert.False(t, segmentMatch("2023/games/game12:t1:test:t21", seg))

	os.Setenv("test", "12,34,56")
	arr := EnvironmentInstance.GetIntArray("test")
	assert.Equal(t, 3, len(arr))
	assert.Equal(t, 12, arr[0])
	assert.Equal(t, 34, arr[1])
	assert.Equal(t, 56, arr[2])
	os.Unsetenv("test")

}
