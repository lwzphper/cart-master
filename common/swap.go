package common

import "encoding/json"

// SwapTo 通过json tag 进行结构体赋值
func SwapTo(request, model interface{}) (err error) {
	// 将数据编码成json字符串
	dataByte, err := json.Marshal(request)
	if err != nil {
		return
	}
	// 将json字符串解码到相应的数据结构
	return json.Unmarshal(dataByte, model)
}
