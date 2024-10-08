package crvorm

import (
	"log/slog"
	"errors"
)

type QueryManyToOne struct {
	AppDb     string `json:"appDb"`
	ModelId   string `json:"modelId"`
}

func (queryManyToOne *QueryManyToOne) mergeResult(res *QueryResult, relatedRes *QueryResult, refField *Field) {
	relatedFieldName := "id"
	fieldName := refField.Field
	//将每一行的结果按照ID分配到不同的记录行上的关联字段上
	//循环结果的每行数据
	for _, relatedRow := range relatedRes.List {
		for _, row := range res.List {
			value := row[fieldName]
			strValue := ""
			switch value.(type) {
			case string:
				strValue = value.(string)
				value = &QueryResult{
					ModelId: *(refField.RelatedModelId),
					Total:   0,
					Value:   &strValue,
					List:    []map[string]interface{}{},
				}
				row[fieldName] = value
			case *QueryResult:
				strValue = *(value.(*QueryResult).Value)
			}

			if strValue == relatedRow[relatedFieldName] {
				value.(*QueryResult).Total += 1
				value.(*QueryResult).List = append(value.(*QueryResult).List, relatedRow)
			}
		}
	}
}

func (queryManyToOne *QueryManyToOne) getFilter(parentList *QueryResult, refField *Field) *map[string]interface{} {
	//多对一字段本身是数据字段，这个字段的值是对应关联表的ID字段的值
	//查询时就是查询关联表ID字段值在当前字段值列表中的记录
	//查询时同时需要合并字段上本身携带的过滤条件
	//首先获取用于过滤的ID列表
	ids := GetFieldValues(parentList, refField.Field)
	if len(ids) == 0 {
		return nil
	}

	inCon := map[string]interface{}{}
	inCon[Op_in] = ids
	inClause := map[string]interface{}{}
	inClause["id"] = inCon
	if refField.Filter == nil {
		return &inClause
	}

	filter := map[string]interface{}{}
	filter[Op_and] = []interface{}{inClause, refField.Filter}
	return &filter
}

func (queryManyToOne *QueryManyToOne) Query(repo DataRepository, parentList *QueryResult, refField *Field) error {
	if refField.RelatedModelId == nil {
		slog.Error("Many2one field must have relatedModelId", "field", refField.Field, "model", queryManyToOne.ModelId)
		return errors.New("Many2one field must have relatedModelId, field:" + refField.Field+" model:"+queryManyToOne.ModelId)
	}

	if refField.Fields == nil {
		slog.Error("Many2one field must have fields", "field", refField.Field, "model", queryManyToOne.ModelId)
		return errors.New("Many2one field must have fields, field:" + refField.Field+" model:"+queryManyToOne.ModelId)
	}

	if len(*refField.Fields) == 0 {
		slog.Error("Many2one field must have fields", "field", refField.Field, "model", queryManyToOne.ModelId)
		return errors.New("Many2one field must have fields, field:" + refField.Field+" model:"+queryManyToOne.ModelId)
	}
	//slog.Info("queryManyToOne", "query", "queryManyToOne","parentList",parentList,"refField",refField)
	filter := queryManyToOne.getFilter(parentList, refField)
	if filter == nil {
		slog.Error("Many2one field filter is nil", "field", refField.Field, "model", queryManyToOne.ModelId)
		return nil
	}

	//执行查询，构造一个新的Query对象进行子表的查询，这样可以实现多层级数据表的递归查询操作
	refQueryParam := &QueryParam{
		ModelId:    *(refField.RelatedModelId),
		Filter:     filter,
		Fields:     refField.Fields,
		Pagination: refField.Pagination,
		AppDb:      queryManyToOne.AppDb,
		Sorter:     refField.Sorter,
	}
	result, err := ExecuteQuery(refQueryParam,repo,false)
	//更新查询结果到父级数据列表中
	if err != nil {
		return err
	}

	queryManyToOne.mergeResult(parentList, result, refField)
	return nil
}
