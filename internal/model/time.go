package model

import (
	"time"
)

// FlexTime 兼容多种时间格式
type FlexTime time.Time

// UnmarshalJSON 自定义解析
func (ft *FlexTime) UnmarshalJSON(data []byte) error {
	// 去掉引号
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// 尝试多种格式
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			*ft = FlexTime(t)
			return nil
		}
	}

	// 都失败了，使用当前时间
	*ft = FlexTime(time.Now())
	return nil
}

// MarshalJSON 序列化
func (ft FlexTime) MarshalJSON() ([]byte, error) {
	t := time.Time(ft)
	return []byte("\"" + t.Format(time.RFC3339) + "\""), nil
}

// Time 转换为 time.Time
func (ft FlexTime) Time() time.Time {
	return time.Time(ft)
}
