# pages-go

基于golang+gorm的通用分页查询插件

# 支持功能

- 指定分页

- 指定orderby规则

- 指定groupby规则

- 指定比较匹配
- 指定模糊查询

# 使用方式

简单示例

```go
package test

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"pages-go"
	"testing"
	"time"
)

// 表实体
type User struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Email     string
	Age       uint8
	CreatedAt time.Time
	UpdatedAt time.Time
	ClassId   int64
}

// 所有的查询条件，支持的查询方式全写在里面
// 类型必须都为string
// 该条件为 查询出age大于指定值的，通过updatedAt排序，通过classId分类
type UserSearch struct {
	Age       string `db_name:"age"  rule:"? > ?" type:"compare"`
	ClassId   string `db_name:"class_id" groupby:"-"`
	UpdatedAt string `db_name:"updated_at" orderby:"0"`
}

// 查询的返回结构体
type UserResp struct {
	Id        uint
	Name      string
	Age       uint8
	UpdatedAt time.Time
}

func initTable() *gorm.DB {
	dsn := "root:123@tcp(127.0.0.1:3306)/testpage?charset=utf8mb4&parseTime=True&loc=Local"
	if db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{}); err != nil {
		panic("数据库连接失败")
	} else {
		//  迁移 schema
		_ = db.AutoMigrate(&User{})
		if res := db.Model(&User{}).First(map[string]interface{}{}); res.RowsAffected != 0 {
			db.Where("1 = 1").Delete(&User{})
			fmt.Println("清空测试表")
		}
		// 插入数据
		user := []User{
			{Name: "A", Age: 10, Email: "A@163.com", ClassId: 100},
			{Name: "AA", Age: 10, Email: "AA@163.com", ClassId: 100},
			{Name: "AAA", Age: 11, Email: "AA@163.com", ClassId: 100},
			{Name: "B", Age: 100, Email: "B@163.com", ClassId: 101},
			{Name: "BB", Age: 101, Email: "BB@163.com", ClassId: 101},
			{Name: "BBB", Age: 110, Email: "BBB@163.com", ClassId: 101},
			{Name: "C", Age: 102, Email: "C@163.com", ClassId: 102},
			{Name: "CC", Age: 102, Email: "CC@163.com", ClassId: 102},
			{Name: "CCC", Age: 112, Email: "CCC@163.com", ClassId: 102},
		}
		_ = db.Create(&user)
		return db
	}
}

func pageExample() {
	db := initTable()

	// 声明查询插件
	var page pages_go.Pages
	//声明分页页数和当前页，默认每页10条，当前第一页
	pageBase := pages_go.PageBase{}
	// 声明查询的基础表实体
	var userModel User
	// 声明查询结果返回的结构体
	var userResp UserResp
	//声明查询条件
	userSearch := UserSearch{
		Age:     "2",
		ClassId: "101",
	}
	// 替换查询条件的值，将会把11替换成1，规则还是查询条件的规则 age>99,updatedAt降序排列
	// 注意： 需要search里有的字段才能生效！！！
	// 该值可能是前端传的，用以替换默认的查询策略
	queryMap := map[string]string{"age": "99", "updatedAt": "1"}

	// 进行插件查询
	err := page.StartPage(db, &pageBase, queryMap, &userSearch, &userModel, &userResp, nil, false, true)
	if err != nil {
		return
	} else {
		//读出查询结果
		myUserResp := page.List.([]*UserResp)
		for _, v := range myUserResp {
			fmt.Println(*v)
		}
	}
}

func TestPage(t *testing.T) {
	pageExample()
}
```