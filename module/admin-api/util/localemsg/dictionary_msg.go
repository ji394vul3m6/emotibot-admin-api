package localemsg

var dictionaryMsg = map[string]map[string]string{
	ZhCn: map[string]string{
		"DictionaryNoClass":                "未分类",
		"DictionaryTemplateXLSXName":       "词库模板",
		"DictionarySheetError":             "获取词库模板资料表错误",
		"DictionaryEmptyRows":              "资料表中无资料",
		"DictionaryErrorEmptyNameTpl":      "行 %d: 词库名为空",
		"DictionaryErrorNameTooLongTpl":    "行 %d: 词库名超过35字",
		"DictionaryErrorSimilarTooLongTpl": "行 %d: 同义词超过64字",
		"DictionaryErrorPathTooLongTpl":    "行 %d: 目录名超过20字",
		"DictionaryErrorRowErrorTpl":       "行 %d：%s",
		"DictionaryErrorPathLevelTpl":      "路径 %d 级内容错误",
		"DictionaryErrorNotEditable":       "该词库不可编辑",
		"DictionaryErrorRequestErrorTpl":   "传入参数有误",
		"DictionaryErrorAPINameTooLong":    "词库名超过35字",
		"DictionaryErrorAPISimilarTooLong": "同义词超过64字",
		"DictionaryErrorAPIPathTooLong":    "目录名超过20字",
		"DictionaryErrorMoveTarget":        "目标目录已有相同名称的词库",
	},
	ZhTw: map[string]string{
		"DictionaryNoClass":                "未分類",
		"DictionaryTemplateXLSXName":       "詞庫模板",
		"DictionarySheetError":             "獲取詞庫模板資料表錯誤",
		"DictionaryEmptyRows":              "資料表中無資料",
		"DictionaryErrorEmptyNameTpl":      "行 %d: 詞庫名為空",
		"DictionaryErrorNameTooLongTpl":    "行 %d: 詞庫名超過35字",
		"DictionaryErrorSimilarTooLongTpl": "行 %d: 同義詞超過64字",
		"DictionaryErrorPathTooLongTpl":    "行 %d: 目錄名超過20字",
		"DictionaryErrorRowErrorTpl":       "行 %d：%s",
		"DictionaryErrorPathLevelTpl":      "路徑 %d 級內容錯誤",
		"DictionaryErrorNotEditable":       "該詞庫不可編輯",
		"DictionaryErrorRequestErrorTpl":   "傳入參數有誤",
		"DictionaryErrorAPINameTooLong":    "詞庫名超過35字",
		"DictionaryErrorAPISimilarTooLong": "同義詞超過64字",
		"DictionaryErrorAPIPathTooLong":    "目錄名超過20字",
		"DictionaryErrorMoveTarget":        "目標目錄已有相同名稱的詞庫",
	},
}
