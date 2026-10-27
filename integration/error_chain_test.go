package integration

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/controllerserr"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services"
)

func TestErrorChainPreservesControllerServiceDataAndSQLCause(t *testing.T) {
	t.Parallel()

	db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.CloseDB()

	classService := services.NewClassService(db, nil)
	_, serviceError := classService.GetClassByClassCode("missing")
	if serviceError == nil {
		t.Fatal("expected missing class error")
	}
	err = controllerserr.NewInternalError("failed to resolve class", serviceError)

	if _, ok := errors.AsType[*controllerserr.InternalError](err); !ok {
		t.Fatal("expected controller error in chain")
	}
	if _, ok := errors.AsType[*serviceerr.ClassCodeNotFoundError](err); !ok {
		t.Fatal("expected service error in chain")
	}
	if _, ok := errors.AsType[*dataerr.NotFoundError](err); !ok {
		t.Fatal("expected data error in chain")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expected sql.ErrNoRows in chain")
	}
}
