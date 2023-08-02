package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
)

func TestPasswordValidate(t *testing.T) {

	var err error

	badpasswords := []PasswordModel{
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "new", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "ebgbmmvizycogyypifbnppywtvjdgkncaxmlhdnfibnmxwhmkvvxokfaaoexgdqnoaainnmuykfhymldalggtehdkhznbvddbztgzovshahgqykqxltmxwlfbagjkwlhpeajfdwfaguvtpalkochtlbpqezltaunhhgoaltoidbzfnrvpqgeyijorhzyqdzvonwscwaomkqlnqjyyljgrwtrcdquehdbqmqraayixjrssmfqojbpmitnwtfeavzieyqiltupeqklbqzrqmmhykhgcknvhwvvshgggxuxgnigaenfjwjmiosfxoeddaygkuonrowwkhoiyazcpuxmpdezjcpjecohagdiuqrkzjheepjrybcqpwpnehdhsdoxvhypxybodjksuekznotwpklkcobdohnzscilvttqjpzfseuvtuqfiyrpcnpxvdfenjifkqdrupmvdrtztbsvvkbgnvincfbmpgvufzghwcgoeggyhoxbwvficizqhutjizrgpqtmgabmhmxluqsetldpjhkmnbtxcxfqcwnezllvycvakgdozncjsnxeotiteuhxyctbflnzrrzlvqqndkictvkhcxjjdkgsheexzyxykidmkbnsdpndxlcpoeepbnywt", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "old", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "ebgbmmvizycogyypifbnppywtvjdgkncaxmlhdnfibnmxwhmkvvxokfaaoexgdqnoaainnmuykfhymldalggtehdkhznbvddbztgzovshahgqykqxltmxwlfbagjkwlhpeajfdwfaguvtpalkochtlbpqezltaunhhgoaltoidbzfnrvpqgeyijorhzyqdzvonwscwaomkqlnqjyyljgrwtrcdquehdbqmqraayixjrssmfqojbpmitnwtfeavzieyqiltupeqklbqzrqmmhykhgcknvhwvvshgggxuxgnigaenfjwjmiosfxoeddaygkuonrowwkhoiyazcpuxmpdezjcpjecohagdiuqrkzjheepjrybcqpwpnehdhsdoxvhypxybodjksuekznotwpklkcobdohnzscilvttqjpzfseuvtuqfiyrpcnpxvdfenjifkqdrupmvdrtztbsvvkbgnvincfbmpgvufzghwcgoeggyhoxbwvficizqhutjizrgpqtmgabmhmxluqsetldpjhkmnbtxcxfqcwnezllvycvakgdozncjsnxeotiteuhxyctbflnzrrzlvqqndkictvkhcxjjdkgsheexzyxykidmkbnsdpndxlcpoeepbnywt", NewPw: "newpassword", NewHashed: []byte("fake")},
	}

	for _, input := range badpasswords {
		err = input.Validate()
		assert.Error(t, err, "An error was expected")
	}

	goodpasswords := []PasswordModel{
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: []byte("fake")},
	}

	for _, input := range goodpasswords {
		err = input.Validate()
		assert.NoError(t, err, "An error was not expected")
	}
}

func TestPasswordIsValid(t *testing.T) {

	badpasswords := []PasswordModel{
		{UID: 0, Name: "test", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 1, Name: "test", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 2, Name: "", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "", NewPw: "newpassword", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "", NewHashed: []byte("fake")},
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: nil},
		{UID: 2, Name: "test", OldPw: "oldpassword", NewPw: "newpassword", NewHashed: []byte("")},
	}

	for _, password := range badpasswords {
		assert.False(t, password.IsValid(), "Should be false")
	}

}

func TestPasswordUpdate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectExec("UPDATE users SET user_password").
		WithArgs([]byte("fake"), 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	password := PasswordModel{
		UID:       2,
		Name:      "test",
		OldPw:     "blah",
		NewPw:     "newpassword",
		NewHashed: []byte("fake"),
	}

	err = password.Update()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestPasswordUpdateInvalid(t *testing.T) {

	var err error

	password := PasswordModel{
		UID:       1,
		NewHashed: []byte("fake"),
	}

	err = password.Update()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("PasswordModel is not valid"), err, "Error should match")
	}
}
