package sqldb

import (
	"fmt"
	"github.com/mpetavy/common"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

type Repository[T any] struct {
	gorm *gorm.DB
}

func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
		return true
	}

	return false
}

func NewRepository[T any](gorm *gorm.DB) (*Repository[T], error) {
	common.DebugFunc()

	return &Repository[T]{
		gorm: gorm,
	}, nil
}

func (repository *Repository[T]) Save(record *T) error {
	common.DebugFunc()

	tx := repository.gorm.Save(record)
	if IsDuplicateKeyError(tx.Error) {
		return ErrDuplicateFound
	}

	if common.Error(tx.Error) {
		return tx.Error
	}

	return nil
}

func (repository *Repository[T]) SaveAll(records []T) error {
	common.DebugFunc()

	tx := repository.gorm.Create(records)
	if IsDuplicateKeyError(tx.Error) {
		return ErrDuplicateFound
	}

	if common.Error(tx.Error) {
		return tx.Error
	}

	return nil
}

func (repository *Repository[T]) Find(id int) (*T, error) {
	common.DebugFunc()

	var record T

	tx := repository.gorm.First(&record, id)
	if tx.Error != nil && strings.Contains(tx.Error.Error(), "not found") {
		return nil, ErrNotFound
	}

	if common.Error(tx.Error) {
		return nil, tx.Error
	}

	return &record, nil
}

func (repository *Repository[T]) Delete(id int) error {
	common.DebugFunc()

	var record T

	tx := repository.gorm.Delete(&record, id)
	if common.Error(tx.Error) {
		return tx.Error
	}

	if tx.RowsAffected != 1 {
		return ErrNotFound
	}

	return nil
}

func (repository *Repository[T]) FindAll(offset int, limit int) ([]T, error) {
	common.DebugFunc()

	var records []T

	tx := repository.gorm.Order("ID")

	if offset > 0 {
		tx = tx.Offset(offset)
	}

	if limit > 0 {
		tx = tx.Limit(limit)
	}

	tx = tx.Find(&records)
	if tx.Error != nil && strings.Contains(tx.Error.Error(), "not found") {
		return nil, ErrNotFound
	}

	if common.Error(tx.Error) {
		return nil, tx.Error
	}

	return records, nil
}

func (repository *Repository[T]) FindFirst(where *WhereTerm) (*T, error) {
	common.DebugFunc()

	var record T

	w, v := where.Build()

	tx := repository.gorm.Where(w, v...).First(&record)
	if tx.Error != nil && strings.Contains(tx.Error.Error(), "not found") {
		return nil, ErrNotFound
	}

	if common.Error(tx.Error) {
		return nil, tx.Error
	}

	return &record, nil
}

func (repository *Repository[T]) Update(record T) error {
	common.DebugFunc()

	tx := repository.gorm.Model(&record).Updates(record)
	if tx.RowsAffected != 1 {
		return ErrNotFound
	}
	if common.Error(tx.Error) {
		return tx.Error
	}

	return nil
}

func SelectOp(fieldname ...string) string {
	return strings.Join(fieldname, ",")
}

const (
	IsIn    = "_IN_"
	BeginOp = "_BEGIN_"
	EndOp   = "_END"
	AndOp   = "_AND_"
	OrOp    = "_OR_"
	IsNull  = "_IS_NULL_"
)

type WhereTerm struct {
	list []WhereItem
}

func NewWhereTerm() *WhereTerm {
	return &WhereTerm{}
}

func (whereTerm *WhereTerm) Where(whereItem WhereItem) *WhereTerm {
	whereTerm.list = append(whereTerm.list, whereItem)

	return whereTerm
}

func (whereTerm *WhereTerm) Begin() *WhereTerm {
	whereTerm.list = append(whereTerm.list, WhereItem{"", BeginOp, nil})

	return whereTerm
}

func (whereTerm *WhereTerm) End() *WhereTerm {
	whereTerm.list = append(whereTerm.list, WhereItem{"", EndOp, nil})

	return whereTerm
}

func (whereTerm *WhereTerm) Or() *WhereTerm {
	whereTerm.list = append(whereTerm.list, WhereItem{"", OrOp, nil})

	return whereTerm
}

func (whereTerm *WhereTerm) And() *WhereTerm {
	whereTerm.list = append(whereTerm.list, WhereItem{"", AndOp, nil})

	return whereTerm
}

func ToAny[T any](arrOrSingle T) []any {
	anys := []any{}

	common.IgnoreError(common.Catch(func() error {
		valueOf := reflect.ValueOf(arrOrSingle)

		switch valueOf.Kind() {
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			for i := 0; i < valueOf.Len(); i++ {
				anys = append(anys, valueOf.Index(i).Interface())
			}
		default:
			anys = append(anys, arrOrSingle)
		}

		return nil
	}))

	return anys
}

type WhereItem struct {
	Fieldname string
	Operator  string
	Value     any
}

func (whereItem WhereItem) Build() (string, []any) {
	var values []any
	sb := strings.Builder{}

	switch whereItem.Operator {
	case IsIn:
		sb.WriteString(fmt.Sprintf("%s in (?)", whereItem.Fieldname))
		values = append(values, whereItem.Value)
	case BeginOp:
		sb.WriteString("(")
	case EndOp:
		sb.WriteString(")")
	case AndOp:
		sb.WriteString(" AND ")
	case OrOp:
		sb.WriteString(" OR ")
	case IsNull:
		sb.WriteString(fmt.Sprintf("%s IS NULL", whereItem.Fieldname))
	default:
		if whereItem.Value != nil {
			values = append(values, whereItem.Value)
		}
		sb.WriteString(fmt.Sprintf("%s %s ?", whereItem.Fieldname, whereItem.Operator))
	}

	return sb.String(), ToAny(whereItem.Value)
}

func (whereTerm WhereTerm) Build() (string, []any) {
	var values []any
	sb := strings.Builder{}

	for _, whereItem := range whereTerm.list {
		w, _ := whereItem.Build()

		sb.WriteString(w)

		switch whereItem.Operator {
		case IsIn:
			values = append(values, whereItem.Value)
		default:
			if whereItem.Value != nil {
				values = append(values, whereItem.Value)
			}
		}
	}

	return sb.String(), values
}

func OrderByOp(fieldname string, ascending bool) string {
	return fmt.Sprintf("%s %s", fieldname, common.Eval(ascending, "asc", "desc"))
}
