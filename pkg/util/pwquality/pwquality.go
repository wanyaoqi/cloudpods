// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pwquality

import (
	"strconv"
	"strings"
	"unicode"

	"yunion.io/x/pkg/errors"
)

// ErrPasswordTooWeak 表示密码强度不符合要求的统一错误
var ErrPasswordTooWeak = errors.Error("password too weak")

// Config 存储 pwquality 配置
type Config struct {
	Minlen   int // 最小长度
	Dcredit  int // 数字字符信用值（负数表示至少需要多少个字符，正数表示每个字符可减少的长度要求）
	Ucredit  int // 大写字母信用值
	Lcredit  int // 小写字母信用值
	Ocredit  int // 特殊字符信用值
	Minclass int // 最小字符类数量（数字、大写、小写、特殊）
}

// HasAnyPolicy 检查配置是否有任何非默认的密码策略设置
// 用于判断配置是否有效（即是否包含任何密码强度要求）
func (c *Config) HasAnyPolicy() bool {
	if c == nil {
		return false
	}
	return c.Minlen > 0 || c.Dcredit != 0 || c.Ucredit != 0 || 
		c.Lcredit != 0 || c.Ocredit != 0 || c.Minclass > 0
}

// ParseConfig 解析 /etc/security/pwquality.conf 配置文件内容
func ParseConfig(content []byte) *Config {
	config := &Config{
		Minlen:   0, // 默认值
		Dcredit:  0, // 默认值
		Ucredit:  0, // 默认值
		Lcredit:  0, // 默认值
		Ocredit:  0, // 默认值
		Minclass: 0, // 默认值
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 key = value 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "minlen":
			if v, err := strconv.Atoi(value); err == nil {
				config.Minlen = v
			}
		case "dcredit":
			if v, err := strconv.Atoi(value); err == nil {
				config.Dcredit = v
			}
		case "ucredit":
			if v, err := strconv.Atoi(value); err == nil {
				config.Ucredit = v
			}
		case "lcredit":
			if v, err := strconv.Atoi(value); err == nil {
				config.Lcredit = v
			}
		case "ocredit":
			if v, err := strconv.Atoi(value); err == nil {
				config.Ocredit = v
			}
		case "minclass":
			if v, err := strconv.Atoi(value); err == nil {
				config.Minclass = v
			}
		}
	}

	return config
}

// Validate 根据 pwquality 配置校验密码强度
// 参考 libpwquality 的实现逻辑
func (c *Config) Validate(password string) error {
	if c == nil {
		// 如果没有配置，不进行校验
		return nil
	}

	// 统计各类字符数量
	var digits, uppers, lowers, others int
	for _, r := range password {
		if unicode.IsDigit(r) {
			digits++
		} else if unicode.IsUpper(r) {
			uppers++
		} else if unicode.IsLower(r) {
			lowers++
		} else {
			others++
		}
	}

	// 处理 credit 值
	// 根据 libpwquality 的文档：
	// - 负数（如 -1）：表示至少需要多少个字符（常用）
	// - 正数（如 1）：表示每个字符可以减少多少长度要求（较少用）
	// - 0：不要求
	if c.Dcredit < 0 {
		// 负数：至少需要这么多数字字符
		required := -c.Dcredit
		if digits < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d digit(s), got %d", required, digits)
		}
	} else if c.Dcredit > 0 {
		// 正数：每个数字字符可以减少多少长度要求（较少使用）
		// 这里我们简化处理，只检查是否有数字
		if digits == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one digit")
		}
	}

	if c.Ucredit < 0 {
		required := -c.Ucredit
		if uppers < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d uppercase letter(s), got %d", required, uppers)
		}
	} else if c.Ucredit > 0 {
		if uppers == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one uppercase letter")
		}
	}

	if c.Lcredit < 0 {
		required := -c.Lcredit
		if lowers < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d lowercase letter(s), got %d", required, lowers)
		}
	} else if c.Lcredit > 0 {
		if lowers == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one lowercase letter")
		}
	}

	if c.Ocredit < 0 {
		required := -c.Ocredit
		if others < required {
			return errors.Wrapf(ErrPasswordTooWeak, "password requires at least %d special character(s), got %d", required, others)
		}
	} else if c.Ocredit > 0 {
		if others == 0 {
			return errors.Wrapf(ErrPasswordTooWeak, "password should contain at least one special character")
		}
	}

	// 计算有效长度（考虑 credit 的正数值，用于减少长度要求）
	effectiveLength := len(password)
	if c.Dcredit > 0 {
		effectiveLength += digits * c.Dcredit
	}
	if c.Ucredit > 0 {
		effectiveLength += uppers * c.Ucredit
	}
	if c.Lcredit > 0 {
		effectiveLength += lowers * c.Lcredit
	}
	if c.Ocredit > 0 {
		effectiveLength += others * c.Ocredit
	}

	// 检查最小长度
	if c.Minlen > 0 && effectiveLength < c.Minlen {
		return errors.Wrapf(ErrPasswordTooWeak, "effective length %d is less than required %d", effectiveLength, c.Minlen)
	}

	// 检查最小字符类数量
	if c.Minclass > 0 {
		classes := 0
		if digits > 0 {
			classes++
		}
		if uppers > 0 {
			classes++
		}
		if lowers > 0 {
			classes++
		}
		if others > 0 {
			classes++
		}
		if classes < c.Minclass {
			return errors.Wrapf(ErrPasswordTooWeak, "requires at least %d character class(es), got %d", c.Minclass, classes)
		}
	}

	return nil
}

// ParsePAMConfig 解析 PAM 配置文件中的密码强度策略
// 支持 pam_pwquality 和 pam_cracklib 模块
func ParsePAMConfig(content []byte) *Config {
	config := &Config{
		Minlen:   0,
		Dcredit:  0,
		Ucredit:  0,
		Lcredit:  0,
		Ocredit:  0,
		Minclass: 0,
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// 查找 password 相关的 PAM 配置行
		// 格式: password requisite pam_pwquality.so retry=3 minlen=8 dcredit=-1 ucredit=-1
		// 或: password requisite pam_cracklib.so retry=3 minlen=8 dcredit=-1 ucredit=-1
		if !strings.Contains(line, "password") {
			continue
		}

		// 检查是否包含 pam_pwquality 或 pam_cracklib
		if !strings.Contains(line, "pam_pwquality") && !strings.Contains(line, "pam_cracklib") {
			continue
		}

		// 解析参数，格式为 key=value
		// 先找到 .so 后面的参数部分
		parts := strings.Fields(line)
		for _, part := range parts {
			if !strings.Contains(part, "=") {
				continue
			}

			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}

			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "minlen":
				if v, err := strconv.Atoi(value); err == nil {
					config.Minlen = v
				}
			case "dcredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Dcredit = v
				}
			case "ucredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Ucredit = v
				}
			case "lcredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Lcredit = v
				}
			case "ocredit":
				if v, err := strconv.Atoi(value); err == nil {
					config.Ocredit = v
				}
			case "minclass":
				if v, err := strconv.Atoi(value); err == nil {
					config.Minclass = v
				}
			case "difok": // pam_cracklib 特有：至少需要多少个字符与旧密码不同
				// 这个参数不影响密码强度校验，可以忽略
			case "retry": // 重试次数，不影响密码强度校验
			}
		}
	}

	return config
}

// ParsePAMConfigSUSE 解析 SUSE 系统的 PAM 配置
// SUSE 可能使用 /etc/security/pam_pwcheck.conf 或 PAM 配置
func ParsePAMConfigSUSE(content []byte) *Config {
	// SUSE 的 pam_pwcheck.conf 格式类似 pwquality.conf
	// 但也可能使用 PAM 配置，先尝试按 pwquality.conf 格式解析
	config := ParseConfig(content)
	
	// 如果解析失败（没有任何策略设置），尝试 PAM 格式
	if !config.HasAnyPolicy() {
		config = ParsePAMConfig(content)
	}
	
	return config
}

// GeneratePassword 根据配置生成符合强度要求的密码
// passwordGenerator 是一个函数，接受长度参数并返回密码
// 如果 passwordGenerator 为 nil，将使用默认的最小长度 12
func (c *Config) GeneratePassword(passwordGenerator func(int) string) string {
	if c == nil || !c.HasAnyPolicy() {
		// 如果没有配置或配置为空，使用默认长度生成
		if passwordGenerator != nil {
			return passwordGenerator(12)
		}
		return ""
	}

	// 计算所需的最小密码长度
	minLength := c.Minlen
	if minLength == 0 {
		minLength = 8 // 默认最小长度
	}

	// 根据 credit 要求计算额外需要的字符数
	requiredChars := 0
	if c.Dcredit < 0 {
		requiredChars += -c.Dcredit
	}
	if c.Ucredit < 0 {
		requiredChars += -c.Ucredit
	}
	if c.Lcredit < 0 {
		requiredChars += -c.Lcredit
	}
	if c.Ocredit < 0 {
		requiredChars += -c.Ocredit
	}

	// 确保长度满足所有要求
	passwordLength := minLength
	if requiredChars > 0 {
		// 至少需要 minLength 和 requiredChars 中的较大值
		if requiredChars > passwordLength {
			passwordLength = requiredChars
		}
		// 再加上一些缓冲，确保有足够的字符满足 minclass 要求
		if c.Minclass > 0 && c.Minclass > 1 {
			passwordLength += c.Minclass
		}
	}

	if passwordGenerator == nil {
		return ""
	}

	// 生成密码并验证，直到符合要求
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		password := passwordGenerator(passwordLength)
		if c.Validate(password) == nil {
			return password
		}
		// 如果不符合要求，增加长度重试
		passwordLength++
	}

	// 如果多次尝试都失败，返回一个较长的密码（应该能满足大部分要求）
	return passwordGenerator(passwordLength)
}

