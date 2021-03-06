package reviewer

import (
	"fmt"
	"testing"

	"github.com/daiguadaidai/blingbling/config"
	"github.com/daiguadaidai/blingbling/parser"
)

func TestDropDatabaseReviewer_Review(t *testing.T) {

	var host string = "10.10.10.12"
	var port int = 3306
	var username string = "HH"
	var password string = "oracle"

	sql := `
		Drop Database db1;
    `

	sqlParser := parser.New()
	stmtNodes, err := sqlParser.Parse(sql, "", "")
	if err != nil {
		fmt.Printf("Syntax Error: %v", err)
	}

	// 循环每一个sql语句进行解析, 并且生成相关审核信息
	dbConfig := config.NewDBConfig(host, port, username, password, "")
	reviewConfig := config.NewReviewConfig()
	reviewMSGs := make([]*ReviewMSG, 0, 1)
	for _, stmtNode := range stmtNodes {
		review := NewReviewer(stmtNode, reviewConfig, dbConfig)
		reviewMSG := review.Review()
		reviewMSGs = append(reviewMSGs, reviewMSG)
	}

	for _, reviewMSG := range reviewMSGs {
		if reviewMSG != nil {
			fmt.Println(reviewMSG.String())
		}
	}
}
