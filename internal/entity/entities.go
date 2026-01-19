package entity

import (
	"encoding/json"
	"time"
)

// AccountSource maps to account_source (account suppliers).
type AccountSource struct {
	ID          int64     `db:"id" json:"id"`
	SourceName  string    `db:"source_name" json:"source_name"`
	SourceType  string    `db:"source_type" json:"source_type"`
	HandlerType string    `db:"handler_type" json:"handler_type"`
	APIURL      *string   `db:"api_url" json:"api_url"`
	APIKey      *string   `db:"api_key" json:"api_key"`
	Priority    int       `db:"priority" json:"priority"`
	AutoRental  int       `db:"auto_rental" json:"auto_rental"`
	Status      int       `db:"status" json:"status"`
	Remark      *string   `db:"remark" json:"remark"`
	CreateTime  time.Time `db:"create_time" json:"create_time"`
	UpdateTime  time.Time `db:"update_time" json:"update_time"`
}

// Product maps to product.
type Product struct {
	ID               int64     `db:"id" json:"id"`
	ProductCode      string    `db:"product_code" json:"product_code"`
	ProductName      string    `db:"product_name" json:"product_name"`
	AccountType      string    `db:"account_type" json:"account_type"`
	Category         *string   `db:"category" json:"category"`
	Icon             *string   `db:"icon" json:"icon"`
	ImageURL         *string   `db:"image_url" json:"image_url"`
	Description      *string   `db:"description" json:"description"`
	Price            float64   `db:"price" json:"price"`
	OriginalPrice    *float64  `db:"original_price" json:"original_price"`
	ValidityDays     int       `db:"validity_days" json:"validity_days"`
	SharedLimit      int       `db:"shared_limit" json:"shared_limit"`
	SalesCount       int       `db:"sales_count" json:"sales_count"`
	AutoDelivery     int       `db:"auto_delivery" json:"auto_delivery"`
	ContactInfo      *string   `db:"contact_info" json:"contact_info"`
	UsageInstruction *string   `db:"usage_instruction" json:"usage_instruction"`
	SortOrder        int       `db:"sort_order" json:"sort_order"`
	Status           int       `db:"status" json:"status"`
	CreateTime       time.Time `db:"create_time" json:"create_time"`
	UpdateTime       time.Time `db:"update_time" json:"update_time"`
	CreateBy         *int64    `db:"create_by" json:"create_by"`
	UpdateBy         *int64    `db:"update_by" json:"update_by"`
}

// ProductSource maps to product_source (product-source mapping).
type ProductSource struct {
	ID            int64           `db:"id" json:"id"`
	ProductID     int64           `db:"product_id" json:"product_id"`
	SourceID      int64           `db:"source_id" json:"source_id"`
	UpstreamURL   *string         `db:"upstream_url" json:"upstream_url"`
	UpstreamParam json.RawMessage `db:"upstream_param" json:"upstream_param"`
	Priority      int             `db:"priority" json:"priority"`
	Weight        int             `db:"weight" json:"weight"`
	CostPrice     *float64        `db:"cost_price" json:"cost_price"`
	Stock         int             `db:"stock" json:"stock"`
	Status        int             `db:"status" json:"status"`
	Remark        *string         `db:"remark" json:"remark"`
	CreateTime    time.Time       `db:"create_time" json:"create_time"`
	UpdateTime    time.Time       `db:"update_time" json:"update_time"`
}

// Account maps to account (account instances).
type Account struct {
	ID uint64 `db:"id" json:"id"`
	//CardNumber       string     `db:"card_number" json:"card_number"`
	AccountEmail     string     `db:"account_email" json:"account_email"`       //没有则为空 临时account
	AccountPassword  *string    `db:"account_password" json:"account_password"` //没有则为空 临时account
	Token            *string    `db:"token" json:"token"`
	ProductID        int64      `db:"product_id" json:"product_id"`
	SourceID         int64      `db:"source_id" json:"source_id"`
	UserID           *uint64    `db:"user_id" json:"user_id"` //没有则为空
	Status           string     `db:"status" json:"status"`
	ExpireDate       *time.Time `db:"expire_date" json:"expire_date"`
	Balance          float64    `db:"balance" json:"balance"`
	LastRechargeTime *time.Time `db:"last_recharge_time" json:"last_recharge_time"`
	Remark           *string    `db:"remark" json:"remark"`
	CreateTime       time.Time  `db:"create_time" json:"create_time"`
	UpdateTime       time.Time  `db:"update_time" json:"update_time"`
}

// 一个小时增加一条记录
type Usage struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountId  uint64    `gorm:"column:account_id" json:"account_id"`
	Consume    float64   `gorm:"column:consume" json:"consume"`        // 对于Balance的消费
	Hour       int       `gorm:"type:int;default:0;index" json:"hour"` // 消费小时 (0-23)，从CreateTime提取
	CreateTime time.Time `gorm:"autoCreateTime" json:"create_time"`    // 精确消费时间
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"update_time"`
}
