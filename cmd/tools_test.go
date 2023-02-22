package cmd

import "testing"
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

}
