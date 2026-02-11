package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// AccountSource 账号供应（存储上游地址）
type AccountSource struct {
	ID            int64           `db:"id" json:"id" gorm:"primaryKey;autoIncrement"`
	SourceName    string          `db:"source_name" json:"source_name" gorm:"type:varchar(100);not null"`
	UpstreamURL   *string         `db:"upstream_url" json:"upstream_url" gorm:"type:varchar(255)"`
	UpstreamToken *string         `db:"upstream_token" json:"upstream_token" gorm:"type:varchar(255)"`
	SourceType    string          `db:"source_type" json:"source_type" gorm:"type:varchar(50);not null"`
	Config        json.RawMessage `db:"config" json:"config" gorm:"type:json"` // 上游地址、API配置等
	Priority      int             `db:"priority" json:"priority" gorm:"default:1"`
	AutoRental    bool            `db:"auto_rental" json:"auto_rental" gorm:"default:false"`
	Status        int             `db:"status" json:"status" gorm:"default:1"`
	Remark        *string         `db:"remark" json:"remark" gorm:"type:text"`
	CreateTime    time.Time       `db:"create_time" json:"create_time" gorm:"autoCreateTime"`
	UpdateTime    time.Time       `db:"update_time" json:"update_time" gorm:"autoUpdateTime"`
}

func (AccountSource) TableName() string {
	return "account_source"
}

// Product 产品
type Product struct {
	ID               int64         `db:"id" json:"id" gorm:"primaryKey;autoIncrement"`
	ProductCode      string        `db:"product_code" json:"product_code" gorm:"type:varchar(100);uniqueIndex;not null"`
	ProductName      string        `db:"product_name" json:"product_name" gorm:"type:varchar(200);not null"`
	AccountType      string        `db:"account_type" json:"account_type" gorm:"type:varchar(100)"`
	Category         *string       `db:"category" json:"category" gorm:"type:varchar(100);index"`
	Group            *string       `db:"group" json:"group" gorm:"type:varchar(100);index"`
	Icon             *string       `db:"icon" json:"icon" gorm:"type:varchar(255)"`
	ImageURL         *string       `db:"image_url" json:"image_url" gorm:"type:varchar(255)"`
	DownStreamURL    *string       `db:"down_stream_url" json:"down_stream_url" gorm:"type:varchar(255)"`
	Description      *string       `db:"description" json:"description" gorm:"type:text"`
	Price            float64       `db:"price" json:"price" gorm:"type:decimal(10,2);default:0"`
	OriginalPrice    *float64      `db:"original_price" json:"original_price" gorm:"type:decimal(10,2)"`
	SalesCount       int           `db:"sales_count" json:"sales_count" gorm:"default:0"`
	ContactInfo      *string       `db:"contact_info" json:"contact_info" gorm:"type:text"`
	UsageInstruction *string       `db:"usage_instruction" json:"usage_instruction" gorm:"type:text"`
	ValidityDays     int           `db:"validity_days" json:"validity_days" gorm:"default:9999"`
	SharedLimit      int           `db:"shared_limit" json:"shared_limit" gorm:"default:0"`
	CostPrice        float64       `db:"cost_price" json:"cost_price" gorm:"type:decimal(10,2);default:0"`
	DefaultBalance   float64       `db:"default_balance" json:"default_balance" gorm:"type:decimal(10,2);default:0"`
	OriginalBalance  float64       `db:"original_balance" json:"original_balance" gorm:"type:decimal(10,2);default:0"`
	Stock            int           `db:"stock" json:"stock" gorm:"default:0"`
	AutoDelivery     bool          `db:"auto_delivery" json:"auto_delivery" gorm:"default:false"`
	SortOrder        int           `db:"sort_order" json:"sort_order" gorm:"default:0"`
	Status           int           `db:"status" json:"status" gorm:"default:1;index"`
	Platforms        PlatformArray `db:"platforms" json:"platforms" gorm:"type:jsonb;default:'[]'"`
	Version          int           `db:"version" json:"version" gorm:"default:0"` // 乐观锁：库存
	CreateTime       time.Time     `db:"create_time" json:"create_time" gorm:"autoCreateTime"`
	UpdateTime       time.Time     `db:"update_time" json:"update_time" gorm:"autoUpdateTime"`
}

func (Product) TableName() string {
	return "product"
}

// ProductPlatform 产品平台关联
type ProductPlatform struct {
	ProductCode string  `db:"product_code" json:"product_code" gorm:"index;not null"`    // 商品在平台的Code
	SKU         *string `db:"sku" json:"sku,omitempty" gorm:"type:varchar(100)"`         // 可选的SKU
	Platform    string  `db:"platform" json:"platform" gorm:"type:varchar(50);not null"` // xianyu/douyin/self
}

func (ProductPlatform) TableName() string {
	return "product_platform"
}

// PlatformArray 平台数组类型，用于存储 Product.Platforms JSON 字段
type PlatformArray []ProductPlatform

// Scan 实现 sql.Scanner 接口
func (p *PlatformArray) Scan(value interface{}) error {
	if value == nil {
		*p = PlatformArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal PlatformArray value: %v", value)
	}
	return json.Unmarshal(bytes, p)
}

// Value 实现 driver.Valuer 接口
func (p PlatformArray) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "[]", nil
	}
	return json.Marshal(p)
}

// AccountSourceProcut 账号供应与产品关系
type AccountSourceProcut struct {
	ID         int64     `db:"id" json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID  int64     `db:"product_id" json:"product_id" gorm:"not null;index"`
	SourceID   int64     `db:"source_id" json:"source_id" gorm:"not null;index"`
	Weight     int       `db:"weight" json:"weight" gorm:"default:0"`
	CreateTime time.Time `db:"create_time" json:"create_time" gorm:"autoCreateTime"`
	UpdateTime time.Time `db:"update_time" json:"update_time" gorm:"autoUpdateTime"`
}

func (AccountSourceProcut) TableName() string {
	return "account_source_procut"
}

// Account 账号实例
type Account struct {
	ID               uint64     `db:"id" json:"id" gorm:"primaryKey;autoIncrement"`
	AccountEmail     string     `db:"account_email" json:"account_email" gorm:"type:varchar(255);index"`
	AccountPassword  *string    `db:"account_password" json:"account_password" gorm:"type:varchar(255)"`
	Token            *string    `db:"token" json:"token" gorm:"type:varchar(255);uniqueIndex"`
	ProductID        int64      `db:"product_id" json:"product_id" gorm:"not null;index"`
	SourceID         int64      `db:"source_id" json:"source_id" gorm:"not null;index"`
	UserID           *uint64    `db:"user_id" json:"user_id" gorm:"index"`
	Status           string     `db:"status" json:"status" gorm:"type:varchar(50);default:'active';index"`
	ExpireDate       *time.Time `db:"expire_date" json:"expire_date" gorm:"index"`
	Balance          float64    `db:"balance" json:"balance" gorm:"type:decimal(10,2);default:0"`
	UsedBalance      float64    `db:"used_balance" json:"used_balance" gorm:"type:decimal(10,2);default:0;comment:'已使用余额'"`
	LastRechargeTime *time.Time `db:"last_recharge_time" json:"last_recharge_time"`
	Remark           *string    `db:"remark" json:"remark" gorm:"type:text"`
	Version          int        `db:"version" json:"version" gorm:"default:0"` // 乐观锁：余额
	UseStatus        int        `db:"use_status" json:"use_status" gorm:"type:tinyint;default:0;index;comment:'使用状态:0-未使用 1-已使用'"`
	StartTime        *time.Time `db:"start_time" json:"start_time" gorm:"index;comment:'开始使用时间'"`
	ExpireDays       int        `db:"expire_days" json:"expire_days" gorm:"default:1;comment:'到期天数'"`
	CreateTime       time.Time  `db:"create_time" json:"create_time" gorm:"autoCreateTime"`
	UpdateTime       time.Time  `db:"update_time" json:"update_time" gorm:"autoUpdateTime"`
}

func (Account) TableName() string {
	return "account"
}

// Usage 使用量记录
type Usage struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID     uint64    `gorm:"column:account_id;not null;index:idx_account_time" json:"account_id"`
	TOKENS        uint64    `gorm:"column:tokens" json:"tokens"`
	Consume       float64   `gorm:"column:consume;type:decimal(10,2);default:0" json:"consume"`
	Model         *string   `gorm:"column:model;type:varchar(100)" json:"model,omitempty"`
	OriginMessage *string   `gorm:"column:origin_message;type:text" json:"origin_message,omitempty"`
	CreateTime    time.Time `gorm:"autoCreateTime;index:idx_account_time" json:"create_time"`
	UpdateTime    time.Time `gorm:"autoUpdateTime" json:"update_time"`
}

func (Usage) TableName() string {
	return "usage"
}

// DailyUsage 按天统计的使用量
type DailyUsage struct {
	Date         string  `json:"date"`          // 日期格式: 2006-01-02
	TotalConsume float64 `json:"total_consume"` // 当天总消费
	RecordCount  int64   `json:"record_count"`  // 记录条数
}

// AccountSourceConfig 供应商配置（存储在 AccountSource.Config JSON 字段中）
type AccountSourceConfig struct {
	APIURL      *string `json:"api_url"` // 上游地址
	APIKey      *string `json:"api_key"`
	HandlerType string  `json:"handler_type"` // 处理器类型
}

// UpstreamErrorLog 上游错误日志
type UpstreamErrorLog struct {
	ID           int64      `db:"id" json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID    int64      `db:"account_id" json:"account_id" gorm:"not null;index:idx_account_id"`
	SourceID     int64      `db:"source_id" json:"source_id" gorm:"not null;index:idx_source_id"`
	SourceType   string     `db:"source_type" json:"source_type" gorm:"type:varchar(50);not null;index:idx_source_type"`
	UpstreamURL  *string    `db:"upstream_url" json:"upstream_url" gorm:"type:varchar(512)"`
	RequestPath  *string    `db:"request_path" json:"request_path" gorm:"type:varchar(512)"`
	StatusCode   *int       `db:"status_code" json:"status_code"`
	ErrorMessage *string    `db:"error_message" json:"error_message" gorm:"type:text"`
	RequestTime  *time.Time `db:"request_time" json:"request_time"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at" gorm:"autoCreateTime;index:idx_created_at"`
}

func (UpstreamErrorLog) TableName() string {
	return "upstream_error_log"
}
