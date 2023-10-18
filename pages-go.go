package pages_go

import (
	"fmt"
	"gorm.io/gorm"
	"pages-go/utils"
	"reflect"
	"strconv"
	"strings"
)

// Pages 分页插件
type Pages struct {
	//分页的起始
	PageBase
	//总的页数
	TotalPageCount int64 `json:"totalPageCount"`
	//总的数据条数
	TotalDataCount int64 `json:"totalDataCount"`
	//起始位置
	FirstIndex int64 `json:"-"`
	//分页数据
	List interface{} `json:"list"`
	//查询条件
	WhereCase string `json:"-"`
	//排序条件
	OrderCase string `json:"-"`
	//分组条件
	GroupCase string `json:"-"`
	//字段
	FiledCase string `json:"-"`
	//查询实体
	Model interface{} `json:"-"`
	//转换实体
	Convert interface{} `json:"-"`

	Join *string `json:"-"`

	//零值或空值时不输出
	Options interface{} `json:"options,omitempty"`
}

// PageBase 定义起始页和每页条数
type PageBase struct {
	//每页的条数
	PageSize int64 `json:"pageSize"`
	//当前页
	CurrentPage int64 `json:"currentPage"`
}

// StartPage 自定义查询字段+查询条件分页
// @sysDB gorm的数据库指针
// @reqPage 分页请求的基础，默认每页10条，从第一页开始
// @querySearchMap 需要orderby,groupby的字段字典
// @search 查询条件，需要传指针
// @model 查询的具体数据表
// @convert 查询结果转换对象，需要传指针。将查询结构转换成此对象返回
// @join 自定义连表条件，需要传指针，nil不连表
// @isGetDelete 是否查询标记为delete的字段
func (page *Pages) StartPage(sysDB *gorm.DB, reqPage *PageBase, querySearchMap map[string]string,
	search interface{}, model interface{}, convert interface{}, join *string, isGetDelete bool, isDebug bool) (err error) {
	//当前页 默认为1
	currentPage := reqPage.CurrentPage
	if currentPage == 0 {
		currentPage = 1
	}
	//每页的条数 默认为 10
	pageSize := reqPage.PageSize
	if pageSize == 0 {
		pageSize = 10
	}
	//计算查询条件
	page.queryTotal(sysDB, querySearchMap, search, model, join, isGetDelete)
	//构造分页对象
	page.queryPage(currentPage, pageSize)
	//计算option字段
	page.setOptions(sysDB, search)
	err = page.setList(sysDB, convert, join, isGetDelete, isDebug)
	//塞入分页数据
	return
}

func (page *Pages) setOptions(sysDB *gorm.DB, search interface{}) {
	val := reflect.ValueOf(search).Elem()
	_options := make(map[string]interface{})
	//遍历所有的字段，计算出所有的options
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		//字段名
		typeFieldName := typeField.Name
		//数据库名
		filedDbName := typeField.Tag.Get("db_name")
		if filedDbName == "" {
			filedDbName = utils.CamelConvert(typeFieldName)
		}
		//判断options是否为空
		fieldOptions := typeField.Tag.Get("options")
		if fieldOptions != "" {
			//初始化字段解析过来的映射与解析后的映射
			optionsSrcMap := make(map[string]string)
			optionsDstMap := make(map[string]string)
			//options转为key-value形式
			if fieldOptions != "-" {
				//逗号分隔
				fieldOptionsByDh := strings.Split(fieldOptions, ",")
				for _, optionValue := range fieldOptionsByDh {
					//冒号分隔
					optionValueByMh := strings.Split(optionValue, ":")
					optionsSrcMap[optionValueByMh[0]] = optionValueByMh[1]
				}
			}
			//计算选项信息
			var dbOptions []string
			sysDB.Model(page.Model).Select(filedDbName).Group(filedDbName).Find(&dbOptions)
			for _, dbOptionItem := range dbOptions {
				//如果分组条件是空的，则跳过
				if dbOptionItem == "" {
					continue
				}
				//判断是否有映射关系，没有的话，数据库查询出来是什么就是什么
				if _, ok := optionsSrcMap[dbOptionItem]; ok {
					// 存在
					optionsDstMap[dbOptionItem] = optionsSrcMap[dbOptionItem]
				} else {
					optionsDstMap[dbOptionItem] = dbOptionItem
				}
			}
			_options[filedDbName] = optionsDstMap
		}
	}
	//这个判断的目的为了当没有options的时候，前端代码就不显示了
	if len(_options) == 0 {
		page.Options = nil
	} else {
		page.Options = _options
	}
}

// 塞入数据
func (page *Pages) setList(sysDB *gorm.DB, convert interface{}, join *string, isGetDelete bool, isDebug bool) (err error) {
	modelType := reflect.TypeOf(convert)
	//动态创建类型
	data := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0).Interface()
	//构建where条件
	page.WhereCase = "1=1" + page.WhereCase
	//动态拼接查询字段
	val := reflect.ValueOf(convert).Elem()
	for i := 0; i < val.NumField(); i++ {
		//获取字段名称
		typeField := val.Type().Field(i)
		typeFieldName := typeField.Name
		dbName := typeField.Tag.Get("db_name")
		var thisField string
		if dbName == "" {
			thisField = utils.CamelConvert(typeFieldName)
		} else {
			//中横线表示查询的时候不拼接该字段
			if dbName == "-" {
				continue
			}
			thisField = dbName + " as " + utils.CamelConvert(typeFieldName)
		}
		if page.FiledCase == "" {
			page.FiledCase += thisField
		} else {
			page.FiledCase += "," + thisField
		}
	}
	db := sysDB
	if isDebug {
		db = sysDB.Debug()
	}

	db = sysDB.Debug().Model(page.Model).Where(page.WhereCase).Limit(int(page.PageSize)).Offset(int((page.CurrentPage - 1) * page.PageSize))
	if isGetDelete {
		db = db.Unscoped()
	}
	//label := common.EmLabel{}
	if page.FiledCase != "" {
		db = db.Select(page.FiledCase)
	}
	if page.OrderCase != "" {
		db = db.Order(page.OrderCase)
	}
	if join != nil {
		db = db.Joins(*join)
	}
	if page.GroupCase != "" {
		db = db.Group(page.GroupCase)
	}
	err = db.Find(&data).Error

	//赋值查询的结果
	if err == nil {
		page.List = data
	}
	return
}

// 计算分页信息
func (page *Pages) queryPage(currentPageReq int64, pageSizeReq int64) {
	//计算起始计数位置
	firstIndex := pageSizeReq * (currentPageReq - 1)
	//计算总的页数
	totalPageCount := (page.TotalDataCount + pageSizeReq - 1) / pageSizeReq
	//分页数据赋值
	page.PageSize = pageSizeReq
	page.CurrentPage = currentPageReq

	page.FirstIndex = firstIndex
	page.TotalPageCount = totalPageCount
}

// 计算分页条件
func (page *Pages) queryTotal(sysDB *gorm.DB, querySearchMap map[string]string, search interface{}, model interface{}, join *string, isGetDelete bool) {
	//定义排序顺序字典
	seqMap := make(map[string]int)
	//定义排序字典
	orderMap := make(map[int]string)
	//拼接where条件
	val := reflect.ValueOf(search).Elem()
	for i := 0; i < val.NumField(); i++ {
		//获取字段名称
		typeField := val.Type().Field(i)
		typeFieldName := typeField.Name
		//根据搜索条件查找值 注意首字母转小写
		nameReq := strings.ToLower(string(typeFieldName[0])) + (typeField.Name)[1:]
		//value := c.DefaultQuery(nameReq, "")
		value, ok := querySearchMap[nameReq]
		if !ok {
			value = ""
		}
		//数据库字段名称
		fileDbName := typeField.Tag.Get("db_name")
		if fileDbName == "" {
			fileDbName = utils.CamelConvert(typeFieldName)
		}
		//拼接分组条件
		if typeField.Tag.Get("groupby") != "" {
			if page.GroupCase == "" {
				page.GroupCase += fileDbName
			} else {
				page.GroupCase += "," + fileDbName
			}
		}
		//获取排序顺序
		if typeField.Tag.Get("sequence") != "" {
			sequence, err := strconv.Atoi(typeField.Tag.Get("sequence"))
			if err != nil {
				sequence = 0
			}
			if value != "" {
				seqReq, err := strconv.Atoi(value)
				if err != nil {
					seqReq = 0
				}
				sequence = seqReq
				seqMap[fileDbName] = sequence
			}
		}
		//拼接排序条件
		if typeField.Tag.Get("orderby") != "" {
			//判断排序顺序是否为空，不为空则根据顺序值写入orderMap里
			if len(seqMap) == 0 {
				orderby, err := strconv.Atoi(typeField.Tag.Get("orderby"))
				if err != nil {
					orderby = 0
				}
				//判断是否前端传值 如果传了则用前端值
				if value != "" {
					orderReq, err := strconv.Atoi(value)
					if err != nil {
						orderReq = 0
					}
					orderby = orderReq
				}
				//小于0 升序，大于0反之
				var thisOrder string
				if orderby <= 0 {
					thisOrder = "asc"
				} else {
					thisOrder = "desc"
				}
				if page.OrderCase == "" {
					page.OrderCase += fileDbName + " " + thisOrder
				} else {
					page.OrderCase += "," + fileDbName + " " + thisOrder
				}
			} else {
				if seqNumber, ok := seqMap[fileDbName]; ok {
					orderby, err := strconv.Atoi(typeField.Tag.Get("orderby"))
					if err != nil {
						orderby = 0
					}
					//判断是否前端传值 如果传了则用前端值
					if value != "" {
						orderReq, err := strconv.Atoi(value)
						if err != nil {
							orderReq = 0
						}
						orderby = orderReq
					}
					//小于0 升序，大于0反之
					var thisOrder string
					if orderby <= 0 {
						thisOrder = "asc"
					} else {
						thisOrder = "desc"
					}
					orderMap[seqNumber] = fileDbName + " " + thisOrder
				}
			}
		}
		//当值为空的时候,则跳过
		if value == "NULL" || value == "" {
			//判断是否为查询条件赋默认值
			typeFieldValue := val.Field(i).Interface().(string)
			if typeFieldValue != "" {
				value = typeFieldValue
			} else {
				continue
			}
		}
		//对查询条件格式化
		switch typeField.Tag.Get("type") {
		//开头模糊
		case "start_with":
			value = fmt.Sprintf("'%s%s'", value, "%")
			//首尾模糊
		case "all_with":
			value = fmt.Sprintf("'%s%s%s'", "%", value, "%") //修改模糊匹配
			//相等比较 数字类型
		case "equals-number":
			value = fmt.Sprintf("%s", value)
		//相等比较 字符类型
		case "equals-string":
			value = fmt.Sprintf("'%s'", value)
			//大小范围比
		case "compare":
			value = fmt.Sprintf("%s", value)
			//in条件
		case "where_in":
			value = fmt.Sprintf("'%s'", strings.Replace(value, ",", "','", -1))
			//非空
		case "is_null":
			value = "is null"
		default:
			value = fmt.Sprintf("'%s%s'", value, "%")
		}
		//获取检索规则 即tag
		ruleValue := typeField.Tag.Get("rule")
		if ruleValue == "" {
			continue
		} else {
			ruleValue = " and " + ruleValue
		}
		//替换?,转为sql的搜索条件 注意字段驼峰转换
		page.WhereCase += utils.Replace(ruleValue, fileDbName, value)
	}

	//获取order中最大key
	var getMaxKey = func(m map[int]string) int {
		i := 0
		var keys []int
		for k := range m {
			keys = append(keys, k)
			i++
		}
		maxKey := keys[0]
		for j := 0; j < len(keys); j++ {
			if maxKey < keys[j] {
				maxKey = keys[j]
			}
		}
		fmt.Println(maxKey)
		return maxKey
	}
	//遍历获取排序条件
	if len(orderMap) > 0 {
		for i := 0; i <= getMaxKey(orderMap); i++ {
			if orderMap[i] != "" {
				if page.OrderCase == "" {
					page.OrderCase += orderMap[i]
				} else {
					page.OrderCase += "," + orderMap[i]
				}
			}
		}
	}
	//子查询db
	db := sysDB.Model(model)
	//计算总条数
	var totalDataCount int64
	//拼接join
	if join != nil {
		db.Joins(*join)
	}
	//拼接查询条件
	if page.WhereCase != "" {
		db.Where("1=1" + page.WhereCase)
	}
	//判断是否有group
	if page.GroupCase != "" {
		db.Group(page.GroupCase)
	}
	//判断是否查询已删除数据
	if isGetDelete {
		db.Unscoped()
	}
	sysDB.Table("(?) as counts", db).Count(&totalDataCount)
	//赋值
	page.TotalDataCount = totalDataCount
	page.Model = model
}
