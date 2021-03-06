package handle

import (
	"encoding/json"
	"fmt"
	"github.com/daiguadaidai/blingbling/parser"
	"github.com/daiguadaidai/blingbling/reviewer"
	"github.com/gorilla/schema"
	"github.com/juju/errors"
	"io/ioutil"
	"net/http"
)

func SqlReviewHandle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	responseReviewData := new(reviewer.ResponseReviewData)
	requestReviewParam, err := GetRequestReviewParam(r) // 获取自定审核参数
	if err != nil {
		reviewMSGs := make([]*reviewer.ReviewMSG, 0, 1)

		reviewMSG := reviewer.NewReivewMSG()
		reviewMSG.HaveError = true
		reviewMSG.AppendMSG(reviewMSG.HaveError, fmt.Sprintf("%v", err))

		reviewMSGs = append(reviewMSGs, reviewMSG)

		responseReviewData.Code = reviewer.REVIEW_CODE_ERROR
		responseReviewData.ReviewMSGs = reviewMSGs
		fmt.Fprintf(w, responseReviewData.ToJson())
		return
	}

	reviewMSGs := StartReview(requestReviewParam)
	responseReviewData.ReviewMSGs = reviewMSGs

	// 修改检测的返回码 Code
	responseReviewData.ResetCode()

	fmt.Fprintf(w, responseReviewData.ToJson())
	return
}

/* 通过http请求的内容获取相关自定义审核参数
Params:
	_request: 请求
*/
func GetRequestReviewParam(_request *http.Request) (*RequestReviewParam, error) {
	switch _request.Method {
	case "POST":
		return GetReviewConfigByPost(_request)
	case "GET":
		return GetReviewConfigByGet(_request)
	default:
		errMSG := fmt.Sprintf("错误请求类型: %v. 值允许使用 GET/POST 请求", _request.Method)
		return nil, errors.New(errMSG)
	}

	// 什么每个逻辑都会返回自己值所以不会走到这里, 所以返回值都是 nil, nil
	return nil, nil
}

// 通过post方法获取自定义审核参数
func GetReviewConfigByPost(_request *http.Request) (*RequestReviewParam, error) {
	bodyBytes, _ := ioutil.ReadAll(_request.Body)
	reviewConfigParam := new(RequestReviewParam)

	if len(bodyBytes) == 0 { // 没有个参数都使用默认值
		return reviewConfigParam, nil
	}

	err := json.Unmarshal(bodyBytes, reviewConfigParam)
	if err != nil {
		errMSG := fmt.Sprintf("POST请求, 不能正确解析给予的值: %v", err)
		return nil, errors.New(errMSG)
	}

	return reviewConfigParam, nil
}

// 通过get方法获取自定义审核参数
func GetReviewConfigByGet(_request *http.Request) (*RequestReviewParam, error) {
	_request.ParseForm()
	reviewConfigParam := new(RequestReviewParam)

	decoder := schema.NewDecoder()
	err := decoder.Decode(reviewConfigParam, _request.URL.Query())
	if err != nil {
		errMSG := fmt.Sprintf("GET请求, 不能正确解析URL参数: %v", err)
		return nil, errors.New(errMSG)
	}

	return reviewConfigParam, nil
}

/* 开始审核 SQL 语句
Params:
    _requestParam: http传来的自定义参数
Return:
	int: 审核状态码
	string: 审核相关信息, 如果成功是成功信息, 如果失败是失败信息
*/
func StartReview(_requestParam *RequestReviewParam) []*reviewer.ReviewMSG {
	reviewConfig := _requestParam.GetReviewConfig() // 获取审核参数
	dbConfig := _requestParam.GetDBConfig()         // 链接数据库配置

	// 循环每一个sql语句进行解析, 并且生成相关审核信息
	reviewMSGs := make([]*reviewer.ReviewMSG, 0, 1)

	// 解析SQL
	sqlParser := parser.New()
	stmtNodes, err := sqlParser.Parse(_requestParam.Sqls, "", "")
	if err != nil {
		reviewMSG := reviewer.NewReivewMSG()
		reviewMSG.HaveError = true
		reviewMSG.Sql = _requestParam.Sqls
		reviewMSG.AppendMSG(reviewMSG.HaveError, fmt.Sprintf("sql语法错误: %v", err))

		reviewMSGs = append(reviewMSGs, reviewMSG)

		return reviewMSGs
	}

	for _, stmtNode := range stmtNodes {
		review := reviewer.NewReviewer(stmtNode, reviewConfig, dbConfig)
		if review == nil {
			reviewMSG := reviewer.NewReivewMSG()
			reviewMSG.HaveError = true
			reviewMSG.Sql = stmtNode.Text()
			reviewMSG.AppendMSG(true, "无法匹配到相关SQL语句类型")
			reviewMSGs = append(reviewMSGs, reviewMSG)
			continue
		}

		reviewMSG := review.Review()
		reviewMSG.Sql = stmtNode.Text()
		reviewMSG.ResetHaveErrorAndWarning()
		reviewMSGs = append(reviewMSGs, reviewMSG)
	}

	return reviewMSGs
}
