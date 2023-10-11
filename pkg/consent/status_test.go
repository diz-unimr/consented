package consent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToString(t *testing.T) {
	assert.Equal(t, Status(Declined).String(), "declined")
	assert.Equal(t, Status(NotAsked).String(), "not-asked")
	assert.Equal(t, Status(Accepted).String(), "accepted")
	assert.Equal(t, Status(Expired).String(), "expired")
}
