package model

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	Total int64       `json:"total"`
	List  interface{} `json:"list"`
}

func Success(data interface{}) Response {
	return Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func Error(message string) Response {
	return Response{
		Code:    -1,
		Message: message,
	}
}

func ErrorWithCode(code int, message string) Response {
	return Response{
		Code:    code,
		Message: message,
	}
}

func PageSuccess(total int64, list interface{}) Response {
	return Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			Total: total,
			List:  list,
		},
	}
}
