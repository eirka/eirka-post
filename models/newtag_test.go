package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestNewTagValidateInput(t *testing.T) {

	var err error

	badtags := []NewTagModel{
		{Ib: 0, Tag: "test", TagType: 1},
		{Ib: 1, Tag: "", TagType: 1},
		{Ib: 1, Tag: "test", TagType: 0},
		{Ib: 1, Tag: "t", TagType: 1},
		{Ib: 1, Tag: "t   ", TagType: 1},
		{Ib: 1, Tag: "tt ", TagType: 1},
		{Ib: 1, Tag: "       t", TagType: 1},
		{Ib: 1, Tag: "ebgbmmvizycogyypifbnppywtvjdgkncaxmlhdnfibnmxwhmkvvxokfaaoexgdqnoaainnmuykfhymldalggtehdkhznbvddbztgzovshahgqykqxltmxwlfbagjkwlhpeajfdwfaguvtpalkochtlbpqezltaunhhgoaltoidbzfnrvpqgeyijorhzyqdzvonwscwaomkqlnqjyyljgrwtrcdquehdbqmqraayixjrssmfqojbpmitnwtfeavzieyqiltupeqklbqzrqmmhykhgcknvhwvvshgggxuxgnigaenfjwjmiosfxoeddaygkuonrowwkhoiyazcpuxmpdezjcpjecohagdiuqrkzjheepjrybcqpwpnehdhsdoxvhypxybodjksuekznotwpklkcobdohnzscilvttqjpzfseuvtuqfiyrpcnpxvdfenjifkqdrupmvdrtztbsvvkbgnvincfbmpgvufzghwcgoeggyhoxbwvficizqhutjizrgpqtmgabmhmxluqsetldpjhkmnbtxcxfqcwnezllvycvakgdozncjsnxeotiteuhxyctbflnzrrzlvqqndkictvkhcxjjdkgsheexzyxykidmkbnsdpndxlcpoeepbnywt", TagType: 1},
	}

	for _, input := range badtags {
		err = input.ValidateInput()
		assert.Error(t, err, "An error was expected")
	}

	goodtag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = goodtag.ValidateInput()
	assert.NoError(t, err, "An error was not expected")

}

func TestNewTagIsValid(t *testing.T) {

	badtags := []NewTagModel{
		{Ib: 0, Tag: "test", TagType: 1},
		{Ib: 1, Tag: "", TagType: 1},
		{Ib: 1, Tag: "test", TagType: 0},
	}

	for _, input := range badtags {
		assert.False(t, input.IsValid(), "Should be false")
	}

	goodtag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	assert.True(t, goodtag.IsValid(), "Should be true")

}

func TestNewTagStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`select count\(1\) from tags`).WillReturnRows(statusrows)

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagStatusDuplicate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`select count\(1\) from tags`).WillReturnRows(statusrows)

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrDuplicateTag, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectExec("INSERT into tags").
		WithArgs("test", 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagPostInvalid(t *testing.T) {

	var err error

	tag := NewTagModel{
		Ib:      0,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("NewTagModel is not valid"), err, "Error should match")
	}

}
