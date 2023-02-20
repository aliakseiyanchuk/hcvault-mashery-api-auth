package mashery_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"yanchuk.nl/hcvault-mashery-api-auth/mashery"
)

func TestParseUserInputDurationDaysFormat(t *testing.T) {
	// Date format
	dur, err := mashery.ParseUserInputDuration("3d")
	assert.Nil(t, err)
	assert.Equal(t, 72, int(dur.Hours()))
}

func TestParseUserInputDurationWeekFormat(t *testing.T) {
	// Week format
	dur, err := mashery.ParseUserInputDuration("2w")
	assert.Nil(t, err)
	assert.Equal(t, 336, int(dur.Hours()))
}

func TestParseUserInputDurationDateFormat(t *testing.T) {
	dest := time.Now().Add(time.Hour * 500)

	// Week format
	dur, err := mashery.ParseUserInputDuration(fmt.Sprintf("%d-%d-%d", dest.Year(), dest.Month(), dest.Day()))
	assert.Nil(t, err)
	assert.True(t, dur.Hours() > 500-24)
}

func TestParseUserInputDurationDateFormatBeforeTime(t *testing.T) {
	_, err := mashery.ParseUserInputDuration("2000-01-01")
	assert.Nil(t, err)
}
