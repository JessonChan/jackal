/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestInsertUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xmpp.NewPresence(from, to, xmpp.UnavailableType)

	user := model.User{Username: "ortuman", Password: "1234", LastPresence: p}

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO users (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(user.Username, user.Password, user.LastPresence.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateUser(&user)
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO users (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(user.Username, user.Password, user.LastPresence.String()).
		WillReturnError(errGeneric)

	err = s.InsertOrUpdateUser(&user)
	require.Equal(t, errGeneric, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestDeleteUser(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_versions (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM private_storage (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM vcards (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM users (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errGeneric)
	mock.ExpectRollback()

	err = s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xmpp.NewPresence(from, to, xmpp.UnavailableType)

	var userColumns = []string{"username", "password", "last_presence", "last_presence_at"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns))

	usr, _ := s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, usr)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow("ortuman", "1234", p.String(), time.Now()))
	_, err := s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").WillReturnError(errGeneric)
	_, err = s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestUserExists(t *testing.T) {
	countColums := []string{"count"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums).AddRow(1))

	ok, err := s.UserExists("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("romeo").
		WillReturnError(errGeneric)
	_, err = s.UserExists("romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}
