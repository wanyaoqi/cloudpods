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
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name: "basic config",
			content: `minlen = 8
dcredit = -1
ucredit = -1
lcredit = -1
ocredit = -1
minclass = 3`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
		},
		{
			name: "config with comments",
			content: `# This is a comment
minlen = 12
# Another comment
dcredit = -2
ucredit = 0
lcredit = -1
ocredit = -1`,
			expected: &Config{
				Minlen:   12,
				Dcredit:  -2,
				Ucredit:  0,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 0,
			},
		},
		{
			name: "empty config",
			content: `# Empty config file
# No settings`,
			expected: &Config{
				Minlen:   0,
				Dcredit:  0,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "config with spaces",
			content: `  minlen = 10  
dcredit  =  -1  
ucredit = -1`,
			expected: &Config{
				Minlen:  10,
				Dcredit: -1,
				Ucredit: -1,
				Lcredit: 0,
				Ocredit: 0,
				Minclass: 0,
			},
		},
		{
			name: "positive credit values",
			content: `minlen = 8
dcredit = 1
ucredit = 1
lcredit = 1`,
			expected: &Config{
				Minlen:  8,
				Dcredit: 1,
				Ucredit: 1,
				Lcredit: 1,
				Ocredit: 0,
				Minclass: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseConfig([]byte(tt.content))
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
			if config.Ucredit != tt.expected.Ucredit {
				t.Errorf("Ucredit = %d, want %d", config.Ucredit, tt.expected.Ucredit)
			}
			if config.Lcredit != tt.expected.Lcredit {
				t.Errorf("Lcredit = %d, want %d", config.Lcredit, tt.expected.Lcredit)
			}
			if config.Ocredit != tt.expected.Ocredit {
				t.Errorf("Ocredit = %d, want %d", config.Ocredit, tt.expected.Ocredit)
			}
			if config.Minclass != tt.expected.Minclass {
				t.Errorf("Minclass = %d, want %d", config.Minclass, tt.expected.Minclass)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		password  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "nil config should pass",
			config: nil,
			password: "anypassword",
			wantError: false,
		},
		{
			name: "minlen check - too short",
			config: &Config{Minlen: 8},
			password: "short",
			wantError: true,
			errorMsg: "effective length",
		},
		{
			name: "minlen check - pass",
			config: &Config{Minlen: 8},
			password: "longpassword",
			wantError: false,
		},
		{
			name: "dcredit negative - require at least 1 digit",
			config: &Config{Dcredit: -1},
			password: "nodigits",
			wantError: true,
			errorMsg: "password requires at least 1 digit(s)",
		},
		{
			name: "dcredit negative - pass with digit",
			config: &Config{Dcredit: -1},
			password: "pass1word",
			wantError: false,
		},
		{
			name: "dcredit negative - require 2 digits",
			config: &Config{Dcredit: -2},
			password: "pass1word",
			wantError: true,
			errorMsg: "password requires at least 2 digit(s)",
		},
		{
			name: "dcredit negative - pass with 2 digits",
			config: &Config{Dcredit: -2},
			password: "pass12word",
			wantError: false,
		},
		{
			name: "ucredit negative - require uppercase",
			config: &Config{Ucredit: -1},
			password: "nouppercase",
			wantError: true,
			errorMsg: "password requires at least 1 uppercase letter(s)",
		},
		{
			name: "ucredit negative - pass with uppercase",
			config: &Config{Ucredit: -1},
			password: "passWord",
			wantError: false,
		},
		{
			name: "lcredit negative - require lowercase",
			config: &Config{Lcredit: -1},
			password: "NOLOWERCASE",
			wantError: true,
			errorMsg: "password requires at least 1 lowercase letter(s)",
		},
		{
			name: "lcredit negative - pass with lowercase",
			config: &Config{Lcredit: -1},
			password: "PASSWORDw",
			wantError: false,
		},
		{
			name: "ocredit negative - require special char",
			config: &Config{Ocredit: -1},
			password: "nospecialchar",
			wantError: true,
			errorMsg: "password requires at least 1 special character(s)",
		},
		{
			name: "ocredit negative - pass with special char",
			config: &Config{Ocredit: -1},
			password: "password@",
			wantError: false,
		},
		{
			name: "minclass - require 3 classes",
			config: &Config{Minclass: 3},
			password: "onlylowercase",
			wantError: true,
			errorMsg: "requires at least 3 character class(es)",
		},
		{
			name: "minclass - pass with 3 classes",
			config: &Config{Minclass: 3},
			password: "Pass1word",
			wantError: false,
		},
		{
			name: "minclass - pass with 4 classes",
			config: &Config{Minclass: 3},
			password: "Pass1@word",
			wantError: false,
		},
		{
			name: "complex config - all requirements",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "Pass1@word",
			wantError: false,
		},
		{
			name: "complex config - missing digit",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "Pass@word",
			wantError: true,
			errorMsg: "password requires at least 1 digit(s)",
		},
		{
			name: "complex config - missing uppercase",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "pass1@word",
			wantError: true,
			errorMsg: "password requires at least 1 uppercase letter(s)",
		},
		{
			name: "complex config - missing lowercase",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "PASS1@WORD",
			wantError: true,
			errorMsg: "password requires at least 1 lowercase letter(s)",
		},
		{
			name: "complex config - missing special char",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "Pass1word",
			wantError: true,
			errorMsg: "password requires at least 1 special character(s)",
		},
		{
			name: "complex config - too short",
			config: &Config{
				Minlen:   12,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "Pass1@wor",
			wantError: true,
			errorMsg: "effective length",
		},
		{
			name: "positive credit - dcredit",
			config: &Config{
				Minlen:  8,
				Dcredit: 1,
			},
			password: "password",
			wantError: true,
			errorMsg: "password should contain at least one digit",
		},
		{
			name: "positive credit - pass with digit",
			config: &Config{
				Minlen:  8,
				Dcredit: 1,
			},
			password: "pass1word",
			wantError: false,
		},
		{
			name: "real world example - strong password",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "MyP@ssw0rd",
			wantError: false,
		},
		{
			name: "real world example - weak password",
			config: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			password: "password",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.password)
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate_CharacterClasses(t *testing.T) {
	config := &Config{Minclass: 4}
	
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"all 4 classes", "Pass1@word", false},
		{"3 classes - missing special", "Pass1word", true},
		{"3 classes - missing digit", "Pass@word", true},
		{"3 classes - missing uppercase", "pass1@word", true},
		{"3 classes - missing lowercase", "PASS1@WORD", true},
		{"2 classes", "password", true},
		{"1 class", "PASSWORD", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Validate(tt.password)
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil for password %q", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v for password %q", err, tt.password)
				}
			}
		})
	}
}

func TestParsePAMConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name: "pam_pwquality config",
			content: `# PAM configuration
auth required pam_unix.so
password requisite pam_pwquality.so retry=3 minlen=8 dcredit=-1 ucredit=-1 lcredit=-1 ocredit=-1 minclass=3
password required pam_unix.so sha512 shadow nullok try_first_pass use_authtok`,
			expected: &Config{
				Minlen:   8,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
		},
		{
			name: "pam_cracklib config",
			content: `# PAM configuration
password requisite pam_cracklib.so retry=3 minlen=12 dcredit=-2 ucredit=-1 lcredit=-1 ocredit=-1
password required pam_unix.so`,
			expected: &Config{
				Minlen:   12,
				Dcredit:  -2,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 0,
			},
		},
		{
			name: "PAM config with comments",
			content: `# This is a comment
password requisite pam_pwquality.so minlen=10 dcredit=-1
# Another comment`,
			expected: &Config{
				Minlen:  10,
				Dcredit: -1,
				Ucredit: 0,
				Lcredit: 0,
				Ocredit: 0,
				Minclass: 0,
			},
		},
		{
			name: "PAM config without password module",
			content: `# No password module
auth required pam_unix.so`,
			expected: &Config{
				Minlen:   0,
				Dcredit:  0,
				Ucredit:  0,
				Lcredit:  0,
				Ocredit:  0,
				Minclass: 0,
			},
		},
		{
			name: "multiple password lines - use first",
			content: `password requisite pam_pwquality.so minlen=8 dcredit=-1
password required pam_unix.so
password optional pam_gnome_keyring.so`,
			expected: &Config{
				Minlen:  8,
				Dcredit: -1,
				Ucredit: 0,
				Lcredit: 0,
				Ocredit: 0,
				Minclass: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParsePAMConfig([]byte(tt.content))
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
			if config.Ucredit != tt.expected.Ucredit {
				t.Errorf("Ucredit = %d, want %d", config.Ucredit, tt.expected.Ucredit)
			}
			if config.Lcredit != tt.expected.Lcredit {
				t.Errorf("Lcredit = %d, want %d", config.Lcredit, tt.expected.Lcredit)
			}
			if config.Ocredit != tt.expected.Ocredit {
				t.Errorf("Ocredit = %d, want %d", config.Ocredit, tt.expected.Ocredit)
			}
			if config.Minclass != tt.expected.Minclass {
				t.Errorf("Minclass = %d, want %d", config.Minclass, tt.expected.Minclass)
			}
		})
	}
}

func TestParsePAMConfigSUSE(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *Config
	}{
		{
			name: "SUSE pam_pwcheck.conf format",
			content: `minlen = 8
dcredit = -1
ucredit = -1
lcredit = -1
ocredit = -1`,
			expected: &Config{
				Minlen:  8,
				Dcredit: -1,
				Ucredit: -1,
				Lcredit: -1,
				Ocredit: -1,
				Minclass: 0,
			},
		},
		{
			name: "SUSE PAM format",
			content: `password requisite pam_pwquality.so minlen=10 dcredit=-1`,
			expected: &Config{
				Minlen:  10,
				Dcredit: -1,
				Ucredit: 0,
				Lcredit: 0,
				Ocredit: 0,
				Minclass: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParsePAMConfigSUSE([]byte(tt.content))
			if config.Minlen != tt.expected.Minlen {
				t.Errorf("Minlen = %d, want %d", config.Minlen, tt.expected.Minlen)
			}
			if config.Dcredit != tt.expected.Dcredit {
				t.Errorf("Dcredit = %d, want %d", config.Dcredit, tt.expected.Dcredit)
			}
		})
	}
}

func TestConfig_HasAnyPolicy(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config",
			config:   &Config{},
			expected: false,
		},
		{
			name:     "only minlen",
			config:   &Config{Minlen: 8},
			expected: true,
		},
		{
			name:     "only dcredit",
			config:   &Config{Dcredit: -1},
			expected: true,
		},
		{
			name:     "only ucredit",
			config:   &Config{Ucredit: -1},
			expected: true,
		},
		{
			name:     "only lcredit",
			config:   &Config{Lcredit: -1},
			expected: true,
		},
		{
			name:     "only ocredit",
			config:   &Config{Ocredit: -1},
			expected: true,
		},
		{
			name:     "only minclass",
			config:   &Config{Minclass: 3},
			expected: true,
		},
		{
			name:     "all fields set",
			config:   &Config{Minlen: 8, Dcredit: -1, Ucredit: -1, Lcredit: -1, Ocredit: -1, Minclass: 3},
			expected: true,
		},
		{
			name:     "only lcredit and ocredit",
			config:   &Config{Lcredit: -1, Ocredit: -1},
			expected: true,
		},
		{
			name:     "only minclass",
			config:   &Config{Minclass: 4},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.HasAnyPolicy()
			if got != tt.expected {
				t.Errorf("HasAnyPolicy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_GeneratePassword(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		passwordGenerator func(int) string
		wantError         bool // 生成的密码是否应该通过验证
		description       string
	}{
		{
			name:   "nil config - should return empty or default password",
			config: nil,
			passwordGenerator: func(length int) string {
				return "defaultpassword"
			},
			wantError:   false,
			description: "nil 配置应该返回默认长度的密码",
		},
		{
			name:   "no policy config - should return default password",
			config: &Config{},
			passwordGenerator: func(length int) string {
				return "defaultpassword"
			},
			wantError:   false,
			description: "没有策略的配置应该返回默认长度的密码",
		},
		{
			name: "minlen only - generate valid password",
			config: &Config{
				Minlen: 8,
			},
			passwordGenerator: func(length int) string {
				// 第一次返回不符合要求的密码，第二次返回符合要求的密码
				if length == 8 {
					return "short" // 太短
				}
				return "longpassword" // 符合要求
			},
			wantError:   false,
			description: "只有最小长度要求，应该生成符合要求的密码",
		},
		{
			name: "dcredit requirement - generate password with digits",
			config: &Config{
				Minlen:  8,
				Dcredit: -1, // 至少需要1个数字
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有数字的密码，第二次返回有数字的密码
				if length == 8 {
					return "nodigits" // 没有数字
				}
				return "pass1word" // 有数字，符合要求
			},
			wantError:   false,
			description: "需要数字字符，应该生成包含数字的密码",
		},
		{
			name: "ucredit requirement - generate password with uppercase",
			config: &Config{
				Minlen:  8,
				Ucredit: -1, // 至少需要1个大写字母
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有大写字母的密码，第二次返回有大写字母的密码
				if length == 8 {
					return "nouppercase" // 没有大写字母
				}
				return "passWord" // 有大写字母，符合要求
			},
			wantError:   false,
			description: "需要大写字母，应该生成包含大写字母的密码",
		},
		{
			name: "lcredit requirement - generate password with lowercase",
			config: &Config{
				Minlen:  8,
				Lcredit: -1, // 至少需要1个小写字母
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有小写字母的密码，第二次返回有小写字母的密码
				if length == 8 {
					return "NOLOWERCASE" // 没有小写字母
				}
				return "PASSWORDw" // 有小写字母，符合要求
			},
			wantError:   false,
			description: "需要小写字母，应该生成包含小写字母的密码",
		},
		{
			name: "ocredit requirement - generate password with special char",
			config: &Config{
				Minlen:  8,
				Ocredit: -1, // 至少需要1个特殊字符
			},
			passwordGenerator: func(length int) string {
				// 第一次返回没有特殊字符的密码，第二次返回有特殊字符的密码
				if length == 8 {
					return "nospecial" // 没有特殊字符
				}
				return "pass@word" // 有特殊字符，符合要求
			},
			wantError:   false,
			description: "需要特殊字符，应该生成包含特殊字符的密码",
		},
		{
			name: "complex requirements - multiple retries",
			config: &Config{
				Minlen:   12,
				Dcredit:  -1,
				Ucredit:  -1,
				Lcredit:  -1,
				Ocredit:  -1,
				Minclass: 3,
			},
			passwordGenerator: func(length int) string {
				// 模拟多次尝试：第一次太短，第二次缺少数字，第三次缺少大写，第四次符合要求
				switch length {
				case 12:
					return "short" // 太短
				case 13:
					return "nouppercase1@" // 缺少大写字母
				case 14:
					return "nolowercase1@A" // 缺少小写字母
				default:
					return "ValidPass12@" // 符合所有要求：12个字符，包含数字、大写、小写、特殊字符
				}
			},
			wantError:   false,
			description: "复杂要求，需要多次重试才能生成符合要求的密码",
		},
		{
			name: "nil passwordGenerator - should return empty",
			config: &Config{
				Minlen: 8,
			},
			passwordGenerator: nil,
			wantError:         false,
			description:       "passwordGenerator 为 nil 时应该返回空字符串",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := tt.config.GeneratePassword(tt.passwordGenerator)

			if tt.passwordGenerator == nil {
				// 如果 passwordGenerator 为 nil，应该返回空字符串
				if password != "" {
					t.Errorf("GeneratePassword() with nil generator = %q, want empty string", password)
				}
				return
			}

			if password == "" {
				t.Errorf("GeneratePassword() returned empty string, want non-empty password")
				return
			}

			// 验证生成的密码是否符合配置要求
			if tt.config != nil && tt.config.HasAnyPolicy() {
				err := tt.config.Validate(password)
				if tt.wantError {
					if err == nil {
						t.Errorf("GeneratePassword() generated password %q should fail validation but passed", password)
					}
				} else {
					if err != nil {
						t.Errorf("GeneratePassword() generated password %q failed validation: %v", password, err)
					}
				}
			}
		})
	}
}

func TestConfig_GeneratePassword_RetryMechanism(t *testing.T) {
	// 测试重试机制：确保在多次尝试后能生成符合要求的密码
	attemptCount := 0
	config := &Config{
		Minlen:  10,
		Dcredit: -2, // 需要至少2个数字
		Ucredit: -1, // 需要至少1个大写字母
		Lcredit: -1, // 需要至少1个小写字母
		Ocredit: -1, // 需要至少1个特殊字符
	}

	passwordGenerator := func(length int) string {
		attemptCount++
		// 前几次生成不符合要求的密码
		switch attemptCount {
		case 1:
			return "short" // 太短
		case 2:
			return "nouppercase12@" // 缺少大写字母
		case 3:
			return "NOLOWERCASE12@" // 缺少小写字母
		case 4:
			return "NoSpecial12" // 缺少特殊字符
		case 5:
			return "ValidPass12@" // 符合所有要求
		default:
			return "ValidPass12@" // 后续都返回符合要求的密码
		}
	}

	password := config.GeneratePassword(passwordGenerator)

	if password == "" {
		t.Fatal("GeneratePassword() returned empty string")
	}

	// 验证密码符合要求
	err := config.Validate(password)
	if err != nil {
		t.Errorf("GeneratePassword() generated password %q failed validation: %v", password, err)
	}

	// 验证确实进行了多次尝试（至少尝试了4次）
	if attemptCount < 4 {
		t.Errorf("Expected at least 4 attempts, got %d", attemptCount)
	}
}

func TestConfig_GeneratePassword_LengthCalculation(t *testing.T) {
	// 测试密码长度计算逻辑
	tests := []struct {
		name           string
		config         *Config
		expectedMinLen int
		description    string
	}{
		{
			name: "minlen only",
			config: &Config{
				Minlen: 8,
			},
			expectedMinLen: 8,
			description:    "只有最小长度要求",
		},
		{
			name: "minlen with credit requirements",
			config: &Config{
				Minlen:  8,
				Dcredit: -2, // 需要2个数字
				Ucredit: -1, // 需要1个大写字母
			},
			expectedMinLen: 8, // 至少是 minlen 和 requiredChars 的较大值
			description:    "最小长度和 credit 要求",
		},
		{
			name: "minlen with minclass",
			config: &Config{
				Minlen:   8,
				Minclass: 3,
			},
			expectedMinLen: 8,
			description:    "最小长度和最小字符类要求",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			passwordGenerator := func(length int) string {
				callCount++
				// 第一次调用时记录长度
				if callCount == 1 {
					if length < tt.expectedMinLen {
						t.Errorf("First password generation length = %d, want at least %d", length, tt.expectedMinLen)
					}
				}
				// 根据配置返回符合要求的密码
				if tt.config.Dcredit < 0 && -tt.config.Dcredit >= 2 {
					// 需要至少2个数字
					return "ValidPass12@"
				}
				return "ValidPass1@"
			}

			password := tt.config.GeneratePassword(passwordGenerator)
			if password == "" {
				t.Errorf("GeneratePassword() returned empty string")
			}

			err := tt.config.Validate(password)
			if err != nil {
				t.Errorf("Generated password failed validation: %v", err)
			}
		})
	}
}

