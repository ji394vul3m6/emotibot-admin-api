package data

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRequestBody      = errors.New("Invalid request body")
	ErrTagTypeNotFound         = errors.New("Tag type not found")
	ErrInvalidAggType          = errors.New("Invalid aggregation type")
	ErrInvalidAggTimeInterval  = errors.New("Invalid aggregtaion time interval")
	ErrExportTaskInProcess     = errors.New("Exporting task is in process")
	ErrExportTaskFailed        = errors.New("Exporting task failed")
	ErrExportTaskEmpty         = errors.New("Empty results, nothing to do")
	ErrExportTaskNotFound      = errors.New("Exporting task not found")
	ErrInvalidFeedbacksType    = errors.New("Invalid feedbacks type")
	ErrFaqCategoryPathNotFound = errors.New("FAQ category path not found")
	ErrFaqRobotTagNotFound     = errors.New("FAQ robot tag not found")
)

const (
	ErrCodeInvalidParameterType = iota
	ErrCodeInvalidParameterStartTime
	ErrCodeInvalidParameterEndTime
	ErrCodeInvalidParameterCategory
	ErrCodeInvalidParameterFilter
	ErrCodeInvalidRequestBody
	ErrCodeInvalidParameterPage
	ErrCodeInvalidParameterExportID
	ErrCodeInvalidParameterDimension
	ErrCodeInvalidParameterPlatform
	ErrCodeInvalidParameterGender
	ErrCodeInvalidParameterTop
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Message: message,
	}
}

type ErrorResponseWithCode struct {
	ErrorResponse
	Code int `json:"code"`
}

func NewErrorResponseWithCode(code int, message string) ErrorResponseWithCode {
	return ErrorResponseWithCode{
		ErrorResponse: ErrorResponse{
			Message: message,
		},
		Code: code,
	}
}

func NewBadRequestResponse(code int, column string) ErrorResponseWithCode {
	message := fmt.Sprintf("Invalid parameter: %s", column)
	return NewErrorResponseWithCode(code, message)
}
