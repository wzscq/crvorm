package crvorm

import (
	"log/slog"
)

type QueryFile struct {
	AppDb     string `json:"appDb"`
	ModelId   string `json:"modelId"`
}

func (queryFile *QueryFile) mergeResult(res *QueryResult, relatedRes *QueryResult, refField *Field) {
	relatedFieldName := "row_id"
	fieldName := refField.Field
	//将每一行的结果按照ID分配到不同的记录行上的关联字段上
	//循环结果的每行数据
	for _, relatedRow := range relatedRes.List {
		for _, row := range res.List {
			//一对多字段,关联表的关联字段存储了本表的ID，
			value, ok := row[fieldName]
			if !ok {
				value = &QueryResult{
					Total:   0,
					ModelId: "core_file",
					List:    []map[string]interface{}{},
				}
				row[fieldName] = value
			}
			slog.Debug("mergeResult", "id", row["id"], "relatedFieldName", relatedRow[relatedFieldName])
			//所以判断本表ID的值和关联表对应关联字段的值是否相等
			if row["id"] == relatedRow[relatedFieldName] {
				value.(*QueryResult).Total += 1
				value.(*QueryResult).List = append(value.(*QueryResult).List, relatedRow)
			}
		}
	}
}

func (queryFile *QueryFile) getFilter(parentList *QueryResult, fileField *Field) *map[string]interface{} {
	//文件表的查询，需要通过model_id,file_id,row_id三个字段来查询
	//首先获取用于过滤的ID列表
	ids := GetFieldValues(parentList, "id")
	rowCon := map[string]interface{}{}
	rowCon[Op_in] = ids
	inClause := map[string]interface{}{}
	inClause["row_id"] = rowCon

	fieldName := fileField.Field
	fieldCon := map[string]interface{}{}
	fieldCon["field_id"] = fieldName

	modelCon := map[string]interface{}{}
	modelCon["model_id"] = queryFile.ModelId

	filter := map[string]interface{}{}

	if fileField.Filter == nil {
		filter[Op_and] = []interface{}{inClause, fieldCon, modelCon}
	} else {
		filter[Op_and] = []interface{}{inClause, fieldCon, modelCon, *fileField.Filter}
	}

	return &filter
}

func (queryFile *QueryFile) getQueryFields() *[]Field {
	fields := []Field{
		Field{
			Field: "id",
		},
		Field{
			Field: "model_id",
		},
		Field{
			Field: "field_id",
		},
		Field{
			Field: "row_id",
		},
		Field{
			Field: "path",
		},
		Field{
			Field: "name",
		},
		Field{
			Field: "ext",
		},
		Field{
			Field: "create_time",
		},
		Field{
			Field: "create_user",
		},
		Field{
			Field: "update_time",
		},
		Field{
			Field: "update_user",
		},
		Field{
			Field: "version",
		},
	}
	return &fields
}

func (queryFile *QueryFile) Query(repo DataRepository, parentList *QueryResult, refField *Field) error {
	filter := queryFile.getFilter(parentList, refField)
	files := queryFile.getQueryFields()

	//执行查询，构造一个新的Query对象进行子表的查询，这样可以实现多层级数据表的递归查询操作
	refQueryParam := &QueryParam{
		ModelId:    "core_file",
		Pagination: refField.Pagination,
		Filter:     filter,
		Fields:     files,
		AppDb:      queryFile.AppDb,
		Sorter:     refField.Sorter,
	}
	result, err := ExecuteQuery(refQueryParam,repo, false)
	//更新查询结果到父级数据列表中
	if err != nil {
		return err
	}

	queryFile.mergeResult(parentList, result, refField)
	return nil
}
